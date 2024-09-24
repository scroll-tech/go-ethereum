package zktrie

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"
)

const (
	// proofFlagsLen is the byte length of the flags in the proof header
	// (first 32 bytes).
	proofFlagsLen = 2
)

var (
	// ErrNodeKeyAlreadyExists is used when a node key already exists.
	ErrInvalidField = errors.New("Key not inside the Finite Field")
	// ErrNodeKeyAlreadyExists is used when a node key already exists.
	ErrNodeKeyAlreadyExists = errors.New("key already exists")
	// ErrKeyNotFound is used when a key is not found in the ZkTrieImpl.
	ErrKeyNotFound = errors.New("key not found in ZkTrieImpl")
	// ErrNodeBytesBadSize is used when the data of a node has an incorrect
	// size and can't be parsed.
	ErrNodeBytesBadSize = errors.New("node data has incorrect size in the DB")
	// ErrReachedMaxLevel is used when a traversal of the MT reaches the
	// maximum level.
	ErrReachedMaxLevel = errors.New("reached maximum level of the merkle tree")
	// ErrInvalidNodeFound is used when an invalid node is found and can't
	// be parsed.
	ErrInvalidNodeFound = errors.New("found an invalid node in the DB")
	// ErrInvalidProofBytes is used when a serialized proof is invalid.
	ErrInvalidProofBytes = errors.New("the serialized proof is invalid")
	// ErrEntryIndexAlreadyExists is used when the entry index already
	// exists in the tree.
	ErrEntryIndexAlreadyExists = errors.New("the entry index already exists in the tree")
	// ErrNotWritable is used when the ZkTrieImpl is not writable and a
	// write function is called
	ErrNotWritable = errors.New("merkle Tree not writable")

	dbKeyRootNode = []byte("currentroot")
)

// ZkTrieImpl is the struct with the main elements of the ZkTrieImpl
type ZkTrieImpl struct {
	lock      sync.RWMutex
	db        ZktrieDatabase
	rootKey   *Hash
	writable  bool
	maxLevels int
	Debug     bool

	dirtyIndex   *big.Int
	dirtyStorage map[Hash]*Node
}

func NewZkTrieImpl(storage ZktrieDatabase, maxLevels int) (*ZkTrieImpl, error) {
	return NewZkTrieImplWithRoot(storage, &HashZero, maxLevels)
}

// NewZkTrieImplWithRoot loads a new ZkTrieImpl. If in the storage already exists one
// will open that one, if not, will create a new one.
func NewZkTrieImplWithRoot(storage ZktrieDatabase, root *Hash, maxLevels int) (*ZkTrieImpl, error) {
	mt := ZkTrieImpl{
		db:           storage,
		maxLevels:    maxLevels,
		writable:     true,
		dirtyIndex:   big.NewInt(0),
		dirtyStorage: make(map[Hash]*Node),
	}
	mt.rootKey = root
	if *root != HashZero {
		_, err := mt.GetNode(mt.rootKey)
		if err != nil {
			return nil, err
		}
	}
	return &mt, nil
}

// Root returns the MerkleRoot
func (mt *ZkTrieImpl) Root() (*Hash, error) {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	// short circuit if there are no nodes to hash
	if mt.dirtyIndex.Cmp(big.NewInt(0)) == 0 {
		return mt.rootKey, nil
	}

	hashedDirtyStorage := make(map[Hash]*Node)
	rootKey, err := mt.calcCommitment(mt.rootKey, hashedDirtyStorage, new(sync.Mutex))
	if err != nil {
		return nil, err
	}

	mt.rootKey = rootKey
	mt.dirtyIndex = big.NewInt(0)
	mt.dirtyStorage = hashedDirtyStorage
	if mt.Debug {
		_, err := mt.getNode(mt.rootKey)
		if err != nil {
			panic(fmt.Errorf("load trie root failed hash %v", mt.rootKey.Bytes()))
		}
	}
	return mt.rootKey, nil
}

// MaxLevels returns the MT maximum level
func (mt *ZkTrieImpl) MaxLevels() int {
	return mt.maxLevels
}

// TryUpdate updates a nodeKey & value into the ZkTrieImpl. Where the `k` determines the
// path from the Root to the Leaf. This also return the updated leaf node
func (mt *ZkTrieImpl) TryUpdate(nodeKey *Hash, vFlag uint32, vPreimage []Byte32) error {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k are valid and fit inside the Finite Field.
	if !CheckBigIntInField(nodeKey.BigInt()) {
		return ErrInvalidField
	}

	newLeafNode := NewLeafNode(nodeKey, vFlag, vPreimage)
	path := getPath(mt.maxLevels, nodeKey[:])

	mt.lock.Lock()
	defer mt.lock.Unlock()

	newRootKey, _, err := mt.addLeaf(newLeafNode, mt.rootKey, 0, path)
	// sanity check
	if err == ErrEntryIndexAlreadyExists {
		panic("Encounter unexpected errortype: ErrEntryIndexAlreadyExists")
	} else if err != nil {
		return err
	}
	if newRootKey != nil {
		mt.rootKey = newRootKey
	}
	return nil
}

// pushLeaf recursively pushes an existing oldLeaf down until its path diverges
// from newLeaf, at which point both leafs are stored, all while updating the
// path. pushLeaf returns the node hash of the parent of the oldLeaf and newLeaf
func (mt *ZkTrieImpl) pushLeaf(newLeaf *Node, oldLeaf *Node, lvl int,
	pathNewLeaf []bool, pathOldLeaf []bool) (*Hash, error) {
	if lvl > mt.maxLevels-2 {
		return nil, ErrReachedMaxLevel
	}
	var newParentNode *Node
	if pathNewLeaf[lvl] == pathOldLeaf[lvl] { // We need to go deeper!
		// notice the node corresponding to return hash is always branch
		nextNodeHash, err := mt.pushLeaf(newLeaf, oldLeaf, lvl+1, pathNewLeaf, pathOldLeaf)
		if err != nil {
			return nil, err
		}
		if pathNewLeaf[lvl] { // go right
			newParentNode = NewParentNode(NodeTypeBranch_1, &HashZero, nextNodeHash)
		} else { // go left
			newParentNode = NewParentNode(NodeTypeBranch_2, nextNodeHash, &HashZero)
		}

		newParentNodeKey := mt.newDirtyNodeKey()
		mt.dirtyStorage[*newParentNodeKey] = newParentNode
		return newParentNodeKey, nil
	}
	oldLeafHash, err := oldLeaf.NodeHash()
	if err != nil {
		return nil, err
	}
	newLeafHash, err := newLeaf.NodeHash()
	if err != nil {
		return nil, err
	}

	if pathNewLeaf[lvl] {
		newParentNode = NewParentNode(NodeTypeBranch_0, oldLeafHash, newLeafHash)
	} else {
		newParentNode = NewParentNode(NodeTypeBranch_0, newLeafHash, oldLeafHash)
	}
	// We can add newLeaf now.  We don't need to add oldLeaf because it's
	// already in the tree.
	mt.dirtyStorage[*newLeafHash] = newLeaf
	newParentNodeKey := mt.newDirtyNodeKey()
	mt.dirtyStorage[*newParentNodeKey] = newParentNode
	return newParentNodeKey, nil
}

// Commit calculates the root for the entire trie and persist all the dirty nodes
func (mt *ZkTrieImpl) Commit() error {
	// force root hash calculation if needed
	if _, err := mt.Root(); err != nil {
		return err
	}

	mt.lock.Lock()
	defer mt.lock.Unlock()

	for key, node := range mt.dirtyStorage {
		if err := mt.db.Put(key[:], node.CanonicalValue()); err != nil {
			return err
		}
	}
	mt.dirtyStorage = make(map[Hash]*Node)
	return mt.db.Put(dbKeyRootNode, append([]byte{byte(DBEntryTypeRoot)}, mt.rootKey[:]...))
}

// addLeaf recursively adds a newLeaf in the MT while updating the path, and returns the key
// of the new added leaf.
func (mt *ZkTrieImpl) addLeaf(newLeaf *Node, currNodeKey *Hash,
	lvl int, path []bool) (*Hash, bool, error) {
	var err error
	if lvl > mt.maxLevels-1 {
		return nil, false, ErrReachedMaxLevel
	}
	n, err := mt.getNode(currNodeKey)
	if err != nil {
		return nil, false, err
	}
	switch n.Type {
	case NodeTypeEmpty_New:
		newLeafHash, err := newLeaf.NodeHash()
		if err != nil {
			return nil, false, err
		}

		mt.dirtyStorage[*newLeafHash] = newLeaf
		return newLeafHash, true, nil
	case NodeTypeLeaf_New:
		newLeafHash, err := newLeaf.NodeHash()
		if err != nil {
			return nil, false, err
		}

		if bytes.Equal(currNodeKey[:], newLeafHash[:]) {
			// do nothing, duplicate entry
			return nil, true, nil
		} else if bytes.Equal(newLeaf.NodeKey.Bytes(), n.NodeKey.Bytes()) {
			// update the existing leaf
			mt.dirtyStorage[*newLeafHash] = newLeaf
			return newLeafHash, true, nil
		}
		newSubTrieRootHash, err := mt.pushLeaf(newLeaf, n, lvl, path, getPath(mt.maxLevels, n.NodeKey[:]))
		return newSubTrieRootHash, false, err
	case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
		// We need to go deeper, continue traversing the tree, left or
		// right depending on path
		branchRight := path[lvl]
		childSubTrieRoot := n.ChildL
		if branchRight {
			childSubTrieRoot = n.ChildR
		}
		newChildSubTrieRoot, isTerminal, err := mt.addLeaf(newLeaf, childSubTrieRoot, lvl+1, path)
		if err != nil {
			return nil, false, err
		}

		// do nothing, if child subtrie was not modified
		if newChildSubTrieRoot == nil {
			return nil, false, nil
		}

		newNodetype := n.Type
		if !isTerminal {
			newNodetype = newNodetype.DeduceUpgradeType(branchRight)
		}

		var newNode *Node
		if branchRight {
			newNode = NewParentNode(newNodetype, n.ChildL, newChildSubTrieRoot)
		} else {
			newNode = NewParentNode(newNodetype, newChildSubTrieRoot, n.ChildR)
		}

		// if current node is already dirty, modify in-place
		// else create a new dirty sub-trie
		newCurTrieRootKey := mt.newDirtyNodeKey()
		mt.dirtyStorage[*newCurTrieRootKey] = newNode
		return newCurTrieRootKey, false, err
	case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
		panic("encounter unsupported deprecated node type")
	default:
		return nil, false, ErrInvalidNodeFound
	}
}

// newDirtyNodeKey increments the dirtyIndex and creates a new dirty node key
func (mt *ZkTrieImpl) newDirtyNodeKey() *Hash {
	mt.dirtyIndex.Add(mt.dirtyIndex, BigOne)
	return NewHashFromBigInt(mt.dirtyIndex)
}

// isDirtyNode returns if the node with the given key is dirty or not
func (mt *ZkTrieImpl) isDirtyNode(nodeKey *Hash) bool {
	_, found := mt.dirtyStorage[*nodeKey]
	return found
}

// calcCommitment calculates the commitment for the given sub trie
func (mt *ZkTrieImpl) calcCommitment(rootKey *Hash, hashedDirtyNodes map[Hash]*Node, commitLock *sync.Mutex) (*Hash, error) {
	if !mt.isDirtyNode(rootKey) {
		return rootKey, nil
	}

	root, err := mt.getNode(rootKey)
	if err != nil {
		return nil, err
	}

	switch root.Type {
	case NodeTypeEmpty:
		return &HashZero, nil
	case NodeTypeLeaf_New:
		// leaves are already hashed, we just need to persist it
		break
	case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
		leftDone := make(chan struct{})
		var leftErr error
		go func() {
			root.ChildL, leftErr = mt.calcCommitment(root.ChildL, hashedDirtyNodes, commitLock)
			close(leftDone)
		}()
		root.ChildR, err = mt.calcCommitment(root.ChildR, hashedDirtyNodes, commitLock)
		if err != nil {
			return nil, err
		}
		<-leftDone
		if leftErr != nil {
			return nil, leftErr
		}
	default:
		return nil, errors.New(fmt.Sprint("unexpected node type", root.Type))
	}

	rootHash, err := root.NodeHash()
	if err != nil {
		return nil, err
	}

	commitLock.Lock()
	defer commitLock.Unlock()
	hashedDirtyNodes[*rootHash] = root
	return rootHash, nil
}

func (mt *ZkTrieImpl) tryGet(nodeKey *Hash) (*Node, error) {

	path := getPath(mt.maxLevels, nodeKey[:])
	var nextKey Hash
	nextKey.Set(mt.rootKey)
	n := new(Node)
	//sanity check
	lastNodeType := NodeTypeBranch_3
	for i := 0; i < mt.maxLevels; i++ {
		err := mt.getNodeTo(&nextKey, n)
		if err != nil {
			return nil, err
		}
		//sanity check
		if i > 0 && n.IsTerminal() {
			if lastNodeType == NodeTypeBranch_3 {
				panic("parent node has invalid type: children are not terminal")
			} else if path[i-1] && lastNodeType == NodeTypeBranch_1 {
				panic("parent node has invalid type: right child is not terminal")
			} else if !path[i-1] && lastNodeType == NodeTypeBranch_2 {
				panic("parent node has invalid type: left child is not terminal")
			}
		}

		lastNodeType = n.Type
		switch n.Type {
		case NodeTypeEmpty_New:
			return NewEmptyNode(), ErrKeyNotFound
		case NodeTypeLeaf_New:
			if bytes.Equal(nodeKey[:], n.NodeKey[:]) {
				return n, nil
			}
			return n, ErrKeyNotFound
		case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			if path[i] {
				nextKey.Set(n.ChildR)
			} else {
				nextKey.Set(n.ChildL)
			}
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter deprecated node types")
		default:
			return nil, ErrInvalidNodeFound
		}
	}

	return nil, ErrReachedMaxLevel
}

// TryGet returns the value for key stored in the trie.
// The value bytes must not be modified by the caller.
// If a node was not found in the database, a MissingNodeError is returned.
func (mt *ZkTrieImpl) TryGet(nodeKey *Hash) ([]byte, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	node, err := mt.tryGet(nodeKey)
	if err == ErrKeyNotFound {
		// according to https://github.com/ethereum/go-ethereum/blob/37f9d25ba027356457953eab5f181c98b46e9988/trie/trie.go#L135
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return node.Data(), nil
}

// Delete removes the specified Key from the ZkTrieImpl and updates the path
// from the deleted key to the Root with the new values.  This method removes
// the key from the ZkTrieImpl, but does not remove the old nodes from the
// key-value database; this means that if the tree is accessed by an old Root
// where the key was not deleted yet, the key will still exist. If is desired
// to remove the key-values from the database that are not under the current
// Root, an option could be to dump all the leafs (using mt.DumpLeafs) and
// import them in a new ZkTrieImpl in a new database (using
// mt.ImportDumpedLeafs), but this will lose all the Root history of the
// ZkTrieImpl
func (mt *ZkTrieImpl) TryDelete(nodeKey *Hash) error {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k is valid and fit inside the Finite Field.
	if !CheckBigIntInField(nodeKey.BigInt()) {
		return ErrInvalidField
	}

	mt.lock.Lock()
	defer mt.lock.Unlock()

	newRootKey, _, err := mt.tryDelete(mt.rootKey, nodeKey, getPath(mt.maxLevels, nodeKey[:]))
	if err != nil {
		return err
	}
	mt.rootKey = newRootKey
	return nil
}

func (mt *ZkTrieImpl) tryDelete(rootKey *Hash, nodeKey *Hash, path []bool) (*Hash, bool, error) {
	root, err := mt.getNode(rootKey)
	if err != nil {
		return nil, false, err
	}

	switch root.Type {
	case NodeTypeEmpty_New:
		return nil, false, ErrKeyNotFound
	case NodeTypeLeaf_New:
		if bytes.Equal(nodeKey[:], root.NodeKey[:]) {
			return &HashZero, true, nil
		}
		return nil, false, ErrKeyNotFound
	case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
		branchRight := path[0]
		childKey, siblingKey := root.ChildL, root.ChildR
		if branchRight {
			childKey, siblingKey = root.ChildR, root.ChildL
		}

		newChildKey, newChildIsTerminal, err := mt.tryDelete(childKey, nodeKey, path[1:])
		if err != nil {
			return nil, false, err
		}

		siblingIsTerminal := root.Type == NodeTypeBranch_0 ||
			(branchRight && root.Type == NodeTypeBranch_1) ||
			(!branchRight && root.Type == NodeTypeBranch_2)

		leftChild, rightChild := newChildKey, siblingKey
		leftIsTerminal, rightIsTerminal := newChildIsTerminal, siblingIsTerminal
		if branchRight {
			leftChild, rightChild = siblingKey, newChildKey
			leftIsTerminal, rightIsTerminal = siblingIsTerminal, newChildIsTerminal
		}

		var newNodeType NodeType
		if leftIsTerminal && rightIsTerminal {
			leftIsEmpty := bytes.Equal(HashZero[:], (*leftChild)[:])
			rightIsEmpty := bytes.Equal(HashZero[:], (*rightChild)[:])

			// if both children are terminal and one of them is empty, prune the root node
			// and send return the non-empty child
			if leftIsEmpty || rightIsEmpty {
				if leftIsEmpty {
					return rightChild, true, nil
				}
				return leftChild, true, nil
			} else {
				newNodeType = NodeTypeBranch_0
			}
		} else if leftIsTerminal {
			newNodeType = NodeTypeBranch_1
		} else if rightIsTerminal {
			newNodeType = NodeTypeBranch_2
		} else {
			newNodeType = NodeTypeBranch_3
		}

		newRootKey := mt.newDirtyNodeKey()
		mt.dirtyStorage[*newRootKey] = NewParentNode(newNodeType, leftChild, rightChild)
		return newRootKey, false, nil
	default:
		panic("encounter unsupported deprecated node type")
	}
}

// GetLeafNode is more underlying method than TryGet, which obtain an leaf node
// or nil if not exist
func (mt *ZkTrieImpl) GetLeafNode(nodeKey *Hash) (*Node, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	n, err := mt.tryGet(nodeKey)
	return n, err
}

// GetNode gets a node by node hash from the MT.  Empty nodes are not stored in the
// tree; they are all the same and assumed to always exist.
// <del>for non exist key, return (NewEmptyNode(), nil)</del>
func (mt *ZkTrieImpl) GetNode(nodeHash *Hash) (*Node, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	return mt.getNode(nodeHash)
}

func (mt *ZkTrieImpl) getNodeTo(nodeHash *Hash, node *Node) error {
	if bytes.Equal(nodeHash[:], HashZero[:]) {
		*node = *NewEmptyNode()
		return nil
	}
	if dirtyNode, found := mt.dirtyStorage[*nodeHash]; found {
		*node = *dirtyNode.Copy()
		return nil
	}
	nBytes, err := mt.db.Get(nodeHash[:])
	if err != nil {
		return err
	}

	return node.SetBytes(nBytes)
}

func (mt *ZkTrieImpl) getNode(nodeHash *Hash) (*Node, error) {
	var n Node
	if err := mt.getNodeTo(nodeHash, &n); err != nil {
		return nil, err
	}
	return &n, nil
}

// getPath returns the binary path, from the root to the leaf.
func getPath(numLevels int, k []byte) []bool {
	path := make([]bool, numLevels)
	for n := 0; n < numLevels; n++ {
		path[n] = TestBit(k[:], uint(n))
	}
	return path
}

// NodeAux contains the auxiliary node used in a non-existence proof.
type NodeAux struct {
	Key   *Hash // Key is the node key
	Value *Hash // Value is the value hash in the node
}

// Proof defines the required elements for a MT proof of existence or
// non-existence.
type Proof struct {
	// existence indicates wether this is a proof of existence or
	// non-existence.
	Existence bool
	// depth indicates how deep in the tree the proof goes.
	depth uint
	// notempties is a bitmap of non-empty Siblings found in Siblings.
	notempties [HashByteLen - proofFlagsLen]byte
	// Siblings is a list of non-empty sibling node hashes.
	Siblings []*Hash
	// NodeInfos is a list of nod types along mpt path
	NodeInfos []NodeType
	// NodeKey record the key of node and path
	NodeKey *Hash
	// NodeAux contains the auxiliary information of the lowest common ancestor
	// node in a non-existence proof.
	NodeAux *NodeAux
}

// BuildZkTrieProof prove uniformed way to turn some data collections into Proof struct
func BuildZkTrieProof(rootHash *Hash, k *big.Int, lvl int, getNode func(key *Hash) (*Node, error)) (*Proof,
	*Node, error) {

	p := &Proof{}
	var siblingHash *Hash

	p.NodeKey = NewHashFromBigInt(k)
	kHash := p.NodeKey
	path := getPath(lvl, kHash[:])

	nextHash := rootHash
	for p.depth = 0; p.depth < uint(lvl); p.depth++ {
		n, err := getNode(nextHash)
		if err != nil {
			return nil, nil, err
		}
		p.NodeInfos = append(p.NodeInfos, n.Type)
		switch n.Type {
		case NodeTypeEmpty_New:
			return p, n, nil
		case NodeTypeLeaf_New:
			if bytes.Equal(kHash[:], n.NodeKey[:]) {
				p.Existence = true
				return p, n, nil
			}
			vHash, err := n.ValueHash()
			// We found a leaf whose entry didn't match hIndex
			p.NodeAux = &NodeAux{Key: n.NodeKey, Value: vHash}
			return p, n, err
		case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			if path[p.depth] {
				nextHash = n.ChildR
				siblingHash = n.ChildL
			} else {
				nextHash = n.ChildL
				siblingHash = n.ChildR
			}
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter deprecated node types")
		default:
			return nil, nil, ErrInvalidNodeFound
		}
		if !bytes.Equal(siblingHash[:], HashZero[:]) {
			SetBitBigEndian(p.notempties[:], p.depth)
			p.Siblings = append(p.Siblings, siblingHash)
		}
	}
	return nil, nil, ErrKeyNotFound

}

// VerifyProof verifies the Merkle Proof for the entry and root.
// nodeHash can be nil when try to verify a nonexistent proof
func VerifyProofZkTrie(rootHash *Hash, proof *Proof, node *Node) bool {
	var nodeHash *Hash
	var err error
	if node == nil {
		if proof.NodeAux != nil {
			nodeHash, err = LeafHash(proof.NodeAux.Key, proof.NodeAux.Value)
		} else {
			nodeHash = &HashZero
		}
	} else {
		nodeHash, err = node.NodeHash()
	}

	if err != nil {
		return false
	}

	rootFromProof, err := proof.rootFromProof(nodeHash, proof.NodeKey)
	if err != nil {
		return false
	}
	return bytes.Equal(rootHash[:], rootFromProof[:])
}

// Verify the proof and calculate the root, nodeHash can be nil when try to verify
// a nonexistent proof
func (proof *Proof) Verify(nodeHash *Hash) (*Hash, error) {
	if proof.Existence {
		if nodeHash == nil {
			return nil, ErrKeyNotFound
		}
		return proof.rootFromProof(nodeHash, proof.NodeKey)
	} else {
		if proof.NodeAux == nil {
			return proof.rootFromProof(&HashZero, proof.NodeKey)
		} else {
			if bytes.Equal(proof.NodeKey[:], proof.NodeAux.Key[:]) {
				return nil, fmt.Errorf("non-existence proof being checked against hIndex equal to nodeAux")
			}
			midHash, err := LeafHash(proof.NodeAux.Key, proof.NodeAux.Value)
			if err != nil {
				return nil, err
			}
			return proof.rootFromProof(midHash, proof.NodeKey)
		}
	}

}

func (proof *Proof) rootFromProof(nodeHash, nodeKey *Hash) (*Hash, error) {
	var err error

	sibIdx := len(proof.Siblings) - 1
	path := getPath(int(proof.depth), nodeKey[:])
	for lvl := int(proof.depth) - 1; lvl >= 0; lvl-- {
		var siblingHash *Hash
		if TestBitBigEndian(proof.notempties[:], uint(lvl)) {
			siblingHash = proof.Siblings[sibIdx]
			sibIdx--
		} else {
			siblingHash = &HashZero
		}
		curType := proof.NodeInfos[lvl]
		if path[lvl] {
			nodeHash, err = NewParentNode(curType, siblingHash, nodeHash).NodeHash()
			if err != nil {
				return nil, err
			}
		} else {
			nodeHash, err = NewParentNode(curType, nodeHash, siblingHash).NodeHash()
			if err != nil {
				return nil, err
			}
		}
	}
	return nodeHash, nil
}

// walk is a helper recursive function to iterate over all tree branches
func (mt *ZkTrieImpl) walk(nodeHash *Hash, f func(*Node)) error {
	n, err := mt.getNode(nodeHash)
	if err != nil {
		return err
	}
	if n.IsTerminal() {
		f(n)
	} else {
		f(n)
		if err := mt.walk(n.ChildL, f); err != nil {
			return err
		}
		if err := mt.walk(n.ChildR, f); err != nil {
			return err
		}
	}
	return nil
}

// Walk iterates over all the branches of a ZkTrieImpl with the given rootHash
// if rootHash is nil, it will get the current RootHash of the current state of
// the ZkTrieImpl.  For each node, it calls the f function given in the
// parameters.  See some examples of the Walk function usage in the
// ZkTrieImpl.go and merkletree_test.go
func (mt *ZkTrieImpl) Walk(rootHash *Hash, f func(*Node)) error {
	var err error
	if rootHash == nil {
		rootHash, err = mt.Root()
		if err != nil {
			return err
		}
	}
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	err = mt.walk(rootHash, f)
	return err
}

// GraphViz uses Walk function to generate a string GraphViz representation of
// the tree and writes it to w
func (mt *ZkTrieImpl) GraphViz(w io.Writer, rootHash *Hash) error {
	if rootHash == nil {
		var err error
		rootHash, err = mt.Root()
		if err != nil {
			return err
		}
	}

	mt.lock.RLock()
	defer mt.lock.RUnlock()

	fmt.Fprintf(w,
		"--------\nGraphViz of the ZkTrieImpl with RootHash "+rootHash.BigInt().String()+"\n")

	fmt.Fprintf(w, `digraph hierarchy {
node [fontname=Monospace,fontsize=10,shape=box]
`)
	cnt := 0
	var errIn error
	err := mt.walk(rootHash, func(n *Node) {
		hash, err := n.NodeHash()
		if err != nil {
			errIn = err
		}
		switch n.Type {
		case NodeTypeEmpty_New:
		case NodeTypeLeaf_New:
			fmt.Fprintf(w, "\"%v\" [style=filled];\n", hash.String())
		case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			lr := [2]string{n.ChildL.String(), n.ChildR.String()}
			emptyNodes := ""
			for i := range lr {
				if lr[i] == "0" {
					lr[i] = fmt.Sprintf("empty%v", cnt)
					emptyNodes += fmt.Sprintf("\"%v\" [style=dashed,label=0];\n", lr[i])
					cnt++
				}
			}
			fmt.Fprintf(w, "\"%v\" -> {\"%v\" \"%v\"}\n", hash.String(), lr[0], lr[1])
			fmt.Fprint(w, emptyNodes)
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter unsupported deprecated node type")
		default:
		}
	})
	fmt.Fprintf(w, "}\n")

	fmt.Fprintf(w,
		"End of GraphViz of the ZkTrieImpl with RootHash "+rootHash.BigInt().String()+"\n--------\n")

	if errIn != nil {
		return errIn
	}
	return err
}

// Copy creates a new independent zkTrie from the given trie
func (mt *ZkTrieImpl) Copy() *ZkTrieImpl {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	// Deep copy in-memory dirty nodes
	newDirtyStorage := make(map[Hash]*Node, len(mt.dirtyStorage))
	for key, dirtyNode := range mt.dirtyStorage {
		newDirtyStorage[key] = dirtyNode.Copy()
	}

	newRootKey := *mt.rootKey
	return &ZkTrieImpl{
		db:           mt.db,
		maxLevels:    mt.maxLevels,
		writable:     mt.writable,
		dirtyIndex:   new(big.Int).Set(mt.dirtyIndex),
		dirtyStorage: newDirtyStorage,
		rootKey:      &newRootKey,
		Debug:        mt.Debug,
	}
}
