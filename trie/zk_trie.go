package trie

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/trie/trienode"
)

const (
	// NodeKeyValidBytes is the number of least significant bytes in the node key
	// that are considered valid to addressing the leaf node, and thus limits the
	// maximum trie depth to NodeKeyValidBytes * 8.
	// We need to truncate the node key because the key is the output of Poseidon
	// hash and the key space doesn't fully occupy the range of power of two. It can
	// lead to an ambiguous bit representation of the key in the finite field
	// causing a soundness issue in the zk circuit.
	NodeKeyValidBytes = 31

	// proofFlagsLen is the byte length of the flags in the proof header
	// (first 32 bytes).
	proofFlagsLen = 2
)

var (
	magicHash     = []byte("THIS IS THE MAGIC INDEX FOR ZKTRIE")
	magicSMTBytes = []byte("THIS IS SOME MAGIC BYTES FOR SMT m1rRXgP2xpDI")

	// ErrNodeKeyAlreadyExists is used when a node key already exists.
	ErrInvalidField = errors.New("Key not inside the Finite Field")
	// ErrNodeKeyAlreadyExists is used when a node key already exists.
	ErrNodeKeyAlreadyExists = errors.New("key already exists")
	// ErrKeyNotFound is used when a key is not found in the ZkTrie.
	ErrKeyNotFound = errors.New("key not found in ZkTrie")
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
	// ErrNotWritable is used when the ZkTrie is not writable and a
	// write function is called
	ErrNotWritable = errors.New("merkle Tree not writable")
)

// ZkTrie is the struct with the main elements of the ZkTrie
type ZkTrie struct {
	lock      sync.RWMutex
	owner     common.Hash
	reader    *trieReader
	rootKey   *Hash
	writable  bool
	maxLevels int
	Debug     bool

	dirtyIndex   *big.Int
	dirtyStorage map[Hash]*Node
}

// NewZkTrie loads a new ZkTrie. If in the storage already exists one
// will open that one, if not, will create a new one.
func NewZkTrie(id *ID, db *Database) (*ZkTrie, error) {
	reader, err := newTrieReader(id.StateRoot, id.Owner, db)
	if err != nil {
		return nil, err
	}

	mt := ZkTrie{
		owner:        id.Owner,
		reader:       reader,
		maxLevels:    NodeKeyValidBytes * 8,
		writable:     true,
		dirtyIndex:   big.NewInt(0),
		dirtyStorage: make(map[Hash]*Node),
	}
	mt.rootKey = NewHashFromBytes(id.Root.Bytes())
	if *mt.rootKey != HashZero {
		_, err := mt.GetNode(mt.rootKey)
		if err != nil {
			return nil, err
		}
	}
	return &mt, nil
}

// Root returns the MerkleRoot
func (mt *ZkTrie) Root() (*Hash, error) {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	return mt.root()
}

func (mt *ZkTrie) root() (*Hash, error) {
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

// Hash returns the root hash of SecureBinaryTrie. It does not write to the
// database and can be used even if the trie doesn't have one.
func (mt *ZkTrie) Hash() common.Hash {
	root, err := mt.Root()
	if err != nil {
		panic("root failed in trie.Hash")
	}
	return common.BytesToHash(root.Bytes())
}

// MaxLevels returns the MT maximum level
func (mt *ZkTrie) MaxLevels() int {
	return mt.maxLevels
}

// TryUpdate updates a nodeKey & value into the ZkTrie. Where the `k` determines the
// path from the Root to the Leaf. This also return the updated leaf node
func (mt *ZkTrie) TryUpdate(key []byte, vFlag uint32, vPreimage []Byte32) error {
	// verify that the ZkTrie is writable
	if !mt.writable {
		return ErrNotWritable
	}

	secureKey, err := ToSecureKey(key)
	if err != nil {
		return err
	}
	nodeKey := NewHashFromBigInt(secureKey)

	// verify that k are valid and fit inside the Finite Field.
	if !CheckBigIntInField(nodeKey.BigInt()) {
		return ErrInvalidField
	}

	newLeafNode := NewLeafNode(nodeKey, vFlag, vPreimage)
	path := getPath(mt.maxLevels, nodeKey[:])

	mt.lock.Lock()
	defer mt.lock.Unlock()

	// todo: save preimage
	// mt.db.UpdatePreimage(key, secureKey)

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

// UpdateStorage updates the storage with the given key and value
func (mt *ZkTrie) UpdateStorage(_ common.Address, key, value []byte) error {
	return mt.TryUpdate(key, 1, []Byte32{*NewByte32FromBytes(value)})
}

// UpdateAccount updates the account with the given address and account
func (mt *ZkTrie) UpdateAccount(address common.Address, acc *types.StateAccount) error {
	value, flag := acc.MarshalFields()
	accValue := make([]Byte32, 0, len(value))
	for _, v := range value {
		accValue = append(accValue, *NewByte32FromBytes(v[:]))
	}
	return mt.TryUpdate(address.Bytes(), flag, accValue)
}

// UpdateContractCode updates the contract code with the given address and code
func (mt *ZkTrie) UpdateContractCode(_ common.Address, _ common.Hash, _ []byte) error {
	return nil
}

// pushLeaf recursively pushes an existing oldLeaf down until its path diverges
// from newLeaf, at which point both leafs are stored, all while updating the
// path. pushLeaf returns the node hash of the parent of the oldLeaf and newLeaf
func (mt *ZkTrie) pushLeaf(newLeaf *Node, oldLeaf *Node, lvl int,
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
func (mt *ZkTrie) Commit(collectLeaf bool) (common.Hash, *trienode.NodeSet, error) {
	mt.lock.Lock()
	defer mt.lock.Unlock()

	nodeset, err := mt.commit(collectLeaf)
	if err != nil {
		return common.Hash{}, nodeset, err
	}

	root, err := mt.root()
	if err != nil {
		return common.Hash{}, nil, err
	}
	return common.BytesToHash(root.Bytes()), nodeset, nil
}

// Commit calculates the root for the entire trie and persist all the dirty nodes
func (mt *ZkTrie) commit(collectLeaf bool) (*trienode.NodeSet, error) {
	// force root hash calculation if needed
	if _, err := mt.root(); err != nil {
		return nil, err
	}

	nodeSet := trienode.NewNodeSet(mt.owner)
	for key, node := range mt.dirtyStorage {
		keyBytes := key.Bytes()
		nodeHash := common.BytesToHash(keyBytes)
		// todo: use proper path instead of hash
		nodeSet.AddNode(keyBytes, trienode.New(nodeHash, node.CanonicalValue()))
		if collectLeaf {
			collectLeafNode := func(childHash *Hash) {
				if childHash != nil {
					if childNode, found := mt.dirtyStorage[*childHash]; found && childNode != nil {
						nodeSet.AddLeaf(nodeHash, childNode.CanonicalValue())
					}
				}
			}
			collectLeafNode(node.ChildL)
			collectLeafNode(node.ChildR)
		}
	}
	mt.dirtyStorage = make(map[Hash]*Node)
	return nodeSet, nil
}

// addLeaf recursively adds a newLeaf in the MT while updating the path, and returns the key
// of the new added leaf.
func (mt *ZkTrie) addLeaf(newLeaf *Node, currNodeKey *Hash,
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
func (mt *ZkTrie) newDirtyNodeKey() *Hash {
	mt.dirtyIndex.Add(mt.dirtyIndex, BigOne)
	return NewHashFromBigInt(mt.dirtyIndex)
}

// isDirtyNode returns if the node with the given key is dirty or not
func (mt *ZkTrie) isDirtyNode(nodeKey *Hash) bool {
	_, found := mt.dirtyStorage[*nodeKey]
	return found
}

// calcCommitment calculates the commitment for the given sub trie
func (mt *ZkTrie) calcCommitment(rootKey *Hash, hashedDirtyNodes map[Hash]*Node, commitLock *sync.Mutex) (*Hash, error) {
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

func (mt *ZkTrie) tryGet(nodeKey *Hash) (*Node, error) {

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
func (mt *ZkTrie) TryGet(key []byte) ([]byte, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	secureK, err := ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	node, err := mt.tryGet(NewHashFromBigInt(secureK))
	if err == ErrKeyNotFound {
		// according to https://github.com/ethereum/go-ethereum/blob/37f9d25ba027356457953eab5f181c98b46e9988/trie/trie.go#L135
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return node.Data(), nil
}

// GetStorage returns the value for key stored in the trie.
func (mt *ZkTrie) GetStorage(_ common.Address, key []byte) ([]byte, error) {
	return mt.TryGet(key)
}

// GetAccount returns the account for the given address.
func (mt *ZkTrie) GetAccount(address common.Address) (*types.StateAccount, error) {
	key := address.Bytes()
	res, err := mt.TryGet(key)
	if res == nil || err != nil {
		return nil, err
	}
	return types.UnmarshalStateAccount(res)
}

// GetKey returns the key for the given hash.
func (mt *ZkTrie) GetKey(hashKey []byte) []byte {
	mt.lock.RLock()
	defer mt.lock.RUnlock()
	return mt.getKey(hashKey)
}

// GetKey returns the key for the given hash.
func (mt *ZkTrie) getKey(hashKey []byte) []byte {
	return nil
}

// Delete removes the specified Key from the ZkTrie and updates the path
// from the deleted key to the Root with the new values.  This method removes
// the key from the ZkTrie, but does not remove the old nodes from the
// key-value database; this means that if the tree is accessed by an old Root
// where the key was not deleted yet, the key will still exist. If is desired
// to remove the key-values from the database that are not under the current
// Root, an option could be to dump all the leafs (using mt.DumpLeafs) and
// import them in a new ZkTrie in a new database (using
// mt.ImportDumpedLeafs), but this will lose all the Root history of the
// ZkTrie
func (mt *ZkTrie) TryDelete(key []byte) error {
	// verify that the ZkTrie is writable
	if !mt.writable {
		return ErrNotWritable
	}

	secureKey, err := ToSecureKey(key)
	if err != nil {
		return err
	}

	nodeKey := NewHashFromBigInt(secureKey)

	// verify that k is valid and fit inside the Finite Field.
	if !CheckBigIntInField(nodeKey.BigInt()) {
		return ErrInvalidField
	}

	mt.lock.Lock()
	defer mt.lock.Unlock()

	//mitigate the create-delete issue: do not delete unexisted key
	if r, _ := mt.tryGet(nodeKey); r == nil {
		return nil
	}

	newRootKey, _, err := mt.tryDelete(mt.rootKey, nodeKey, getPath(mt.maxLevels, nodeKey[:]))
	if err != nil {
		return err
	}
	mt.rootKey = newRootKey
	return nil
}

func (mt *ZkTrie) tryDelete(rootKey *Hash, nodeKey *Hash, path []bool) (*Hash, bool, error) {
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

// DeleteAccount removes the account with the given address from the trie.
func (mt *ZkTrie) DeleteAccount(address common.Address) error {
	return mt.TryDelete(address.Bytes())
}

// DeleteStorage removes the key from the trie.
func (mt *ZkTrie) DeleteStorage(_ common.Address, key []byte) error {
	return mt.TryDelete(key)
}

// GetLeafNode is more underlying method than TryGet, which obtain an leaf node
// or nil if not exist
func (mt *ZkTrie) GetLeafNode(key []byte) (*Node, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	secureKey, err := ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	nodeKey := NewHashFromBigInt(secureKey)

	n, err := mt.tryGet(nodeKey)
	return n, err
}

// GetNode gets a node by node hash from the MT.  Empty nodes are not stored in the
// tree; they are all the same and assumed to always exist.
// <del>for non exist key, return (NewEmptyNode(), nil)</del>
func (mt *ZkTrie) GetNode(nodeHash *Hash) (*Node, error) {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	return mt.getNode(nodeHash)
}

func (mt *ZkTrie) getNodeTo(nodeHash *Hash, node *Node) error {
	if bytes.Equal(nodeHash[:], HashZero[:]) {
		*node = *NewEmptyNode()
		return nil
	}
	if dirtyNode, found := mt.dirtyStorage[*nodeHash]; found {
		*node = *dirtyNode.Copy()
		return nil
	}

	var hash common.Hash
	hash.SetBytes(nodeHash.Bytes())
	nBytes, err := mt.reader.node(nil, hash)
	if err != nil {
		return err
	}
	return node.SetBytes(nBytes)
}

func (mt *ZkTrie) getNode(nodeHash *Hash) (*Node, error) {
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
func (mt *ZkTrie) walk(nodeHash *Hash, f func(*Node)) error {
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

// Walk iterates over all the branches of a ZkTrie with the given rootHash
// if rootHash is nil, it will get the current RootHash of the current state of
// the ZkTrie.  For each node, it calls the f function given in the
// parameters.  See some examples of the Walk function usage in the
// ZkTrie.go and merkletree_test.go
func (mt *ZkTrie) Walk(rootHash *Hash, f func(*Node)) error {
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
func (mt *ZkTrie) GraphViz(w io.Writer, rootHash *Hash) error {
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
		"--------\nGraphViz of the ZkTrie with RootHash "+rootHash.BigInt().String()+"\n")

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
		"End of GraphViz of the ZkTrie with RootHash "+rootHash.BigInt().String()+"\n--------\n")

	if errIn != nil {
		return errIn
	}
	return err
}

// Copy creates a new independent zkTrie from the given trie
func (mt *ZkTrie) Copy() *ZkTrie {
	mt.lock.RLock()
	defer mt.lock.RUnlock()

	// Deep copy in-memory dirty nodes
	newDirtyStorage := make(map[Hash]*Node, len(mt.dirtyStorage))
	for key, dirtyNode := range mt.dirtyStorage {
		newDirtyStorage[key] = dirtyNode.Copy()
	}

	newRootKey := *mt.rootKey
	return &ZkTrie{
		reader:       mt.reader,
		maxLevels:    mt.maxLevels,
		writable:     mt.writable,
		dirtyIndex:   new(big.Int).Set(mt.dirtyIndex),
		dirtyStorage: newDirtyStorage,
		rootKey:      &newRootKey,
		Debug:        mt.Debug,
	}
}

// Prove constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
// func (t *ZkTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.KeyValueWriter) error {
func (mt *ZkTrie) Prove(key []byte, proofDb ethdb.KeyValueWriter) error {
	fromLevel := uint(0)
	err := mt.ProveWithDeletion(key, fromLevel, func(n *Node) error {
		nodeHash, err := n.NodeHash()
		if err != nil {
			return err
		}

		if n.Type == NodeTypeLeaf_New {
			preImage := mt.getKey(n.NodeKey.Bytes())
			if len(preImage) > 0 {
				n.KeyPreimage = &Byte32{}
				copy(n.KeyPreimage[:], preImage)
			}
		}
		return proofDb.Put(nodeHash[:], n.Value())
	}, nil)
	if err != nil {
		return err
	}

	// we put this special kv pair in db so we can distinguish the type and
	// make suitable Proof
	return proofDb.Put(magicHash, magicSMTBytes)
}

// DecodeProof try to decode a node bytes, return can be nil for any non-node data (magic code)
func DecodeSMTProof(data []byte) (*Node, error) {

	if bytes.Equal(magicSMTBytes, data) {
		//skip magic bytes node
		return nil, nil
	}

	return NewNodeFromBytes(data)
}

// ProveWithDeletion constructs a merkle proof for key. The result contains all encoded nodes
// on the path to the value at key. The value itself is also included in the last
// node and can be retrieved by verifying the proof.
//
// If the trie does not contain a value for key, the returned proof contains all
// nodes of the longest existing prefix of the key (at least the root node), ending
// with the node that proves the absence of the key.
//
// If the trie contain value for key, the onHit is called BEFORE writeNode being called,
// both the hitted leaf node and its sibling node is provided as arguments so caller
// would receive enough information for launch a deletion and calculate the new root
// base on the proof data
// Also notice the sibling can be nil if the trie has only one leaf
func (mt *ZkTrie) ProveWithDeletion(key []byte, fromLevel uint, writeNode func(*Node) error, onHit func(*Node, *Node)) error {
	secureKey, err := ToSecureKey(key)
	if err != nil {
		return err
	}

	nodeKey := NewHashFromBigInt(secureKey)
	var prev *Node
	return mt.prove(nodeKey, fromLevel, func(n *Node) (err error) {
		defer func() {
			if err == nil {
				err = writeNode(n)
			}
			prev = n
		}()

		if prev != nil {
			switch prev.Type {
			case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			default:
				// sanity check: we should stop after obtain leaf/empty
				panic("unexpected behavior in prove")
			}
		}

		if onHit == nil {
			return
		}

		// check and call onhit
		if n.Type == NodeTypeLeaf_New && bytes.Equal(n.NodeKey.Bytes(), nodeKey.Bytes()) {
			if prev == nil {
				// for sole element trie
				onHit(n, nil)
			} else {
				var sibling, nHash *Hash
				nHash, err = n.NodeHash()
				if err != nil {
					return
				}

				if bytes.Equal(nHash.Bytes(), prev.ChildL.Bytes()) {
					sibling = prev.ChildR
				} else {
					sibling = prev.ChildL
				}

				if siblingNode, err := mt.getNode(sibling); err == nil {
					onHit(n, siblingNode)
				} else {
					onHit(n, nil)
				}
			}

		}
		return
	})
}

// Prove constructs a merkle proof for SMT, it respect the protocol used by the ethereum-trie
// but save the node data with a compact form
func (mt *ZkTrie) prove(kHash *Hash, fromLevel uint, writeNode func(*Node) error) error {
	// force root hash calculation if needed
	if _, err := mt.Root(); err != nil {
		return err
	}

	mt.lock.RLock()
	defer mt.lock.RUnlock()

	path := getPath(mt.maxLevels, kHash[:])
	var nodes []*Node
	var lastN *Node
	tn := mt.rootKey
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.getNode(tn)
		if err != nil {
			fmt.Println("get node fail", err, tn.Hex(),
				lastN.ChildL.Hex(),
				lastN.ChildR.Hex(),
				path,
				i,
			)
			return err
		}
		nodeHash := tn
		lastN = n

		finished := true
		switch n.Type {
		case NodeTypeEmpty_New:
		case NodeTypeLeaf_New:
			// notice even we found a leaf whose entry didn't match the expected k,
			// we still include it as the proof of absence
		case NodeTypeBranch_0, NodeTypeBranch_1, NodeTypeBranch_2, NodeTypeBranch_3:
			finished = false
			if path[i] {
				tn = n.ChildR
			} else {
				tn = n.ChildL
			}
		case NodeTypeEmpty, NodeTypeLeaf, NodeTypeParent:
			panic("encounter deprecated node types")
		default:
			return ErrInvalidNodeFound
		}

		nCopy := n.Copy()
		nCopy.nodeHash = nodeHash
		nodes = append(nodes, nCopy)
		if finished {
			break
		}
	}

	for _, n := range nodes {
		if fromLevel > 0 {
			fromLevel--
			continue
		}

		// TODO: notice here we may have broken some implicit on the proofDb:
		// the key is not kecca(value) and it even can not be derived from
		// the value by any means without a actually decoding
		if err := writeNode(n); err != nil {
			return err
		}
	}

	return nil
}

// NodeIterator returns an iterator that returns nodes of the trie. Iteration
// starts at the key after the given start key. And error will be returned
// if fails to create node iterator.
func (mt *ZkTrie) NodeIterator(start []byte) (NodeIterator, error) {
	return nil, errors.New("not implemented")
}

// VerifyProof checks merkle proofs. The given proof must contain the value for
// key in a trie with the given root hash. VerifyProof returns an error if the
// proof contains invalid trie nodes or the wrong value.
func VerifyProofSMT(rootHash common.Hash, key []byte, proofDb ethdb.KeyValueReader) (value []byte, err error) {
	h := NewHashFromBytes(rootHash.Bytes())
	k, err := ToSecureKey(key)
	if err != nil {
		return nil, err
	}

	proof, n, err := BuildZkTrieProof(h, k, len(key)*8, func(key *Hash) (*Node, error) {
		buf, _ := proofDb.Get(key[:])
		if buf == nil {
			return nil, ErrKeyNotFound
		}
		n, err := NewNodeFromBytes(buf)
		return n, err
	})

	if err != nil {
		// do not contain the key
		return nil, err
	} else if !proof.Existence {
		return nil, nil
	}

	if VerifyProofZkTrie(h, proof, n) {
		return n.Data(), nil
	} else {
		return nil, fmt.Errorf("bad proof node %v", proof)
	}
}
