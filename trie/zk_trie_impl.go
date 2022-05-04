package trie

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"

	"github.com/scroll-tech/go-ethereum/log"

	cryptoUtils "github.com/iden3/go-iden3-crypto/utils"

	"github.com/scroll-tech/go-ethereum/common"
	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
)

const (
	// proofFlagsLen is the byte length of the flags in the proof header
	// (first 32 bytes).
	proofFlagsLen = 2
)

var (
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
	ErrNotWritable = errors.New("Merkle Tree not writable")

	dbKeyRootNode = []byte("currentroot")
)

// ZkTrieImpl is the struct with the main elements of the ZkTrieImpl
type ZkTrieImpl struct {
	db        *ZktrieDatabase
	rootKey   *zkt.Hash
	writable  bool
	maxLevels int
	Debug     bool
}

func NewZkTrieImpl(storage *ZktrieDatabase, maxLevels int) (*ZkTrieImpl, error) {
	return NewZkTrieImplWithRoot(storage, &zkt.HashZero, maxLevels)
}

// NewZkTrieImplWithRoot loads a new ZkTrieImpl. If in the storage already exists one
// will open that one, if not, will create a new one.
func NewZkTrieImplWithRoot(storage *ZktrieDatabase, root *zkt.Hash, maxLevels int) (*ZkTrieImpl, error) {
	mt := ZkTrieImpl{db: storage, maxLevels: maxLevels, writable: true}
	mt.rootKey = root
	if *root != zkt.HashZero {
		_, err := mt.GetNode(mt.rootKey)
		if err != nil {
			return nil, err
		}
	}
	return &mt, nil
}

// DB returns the ZkTrieImpl.DB()
func (mt *ZkTrieImpl) DB() *ZktrieDatabase {
	return mt.db
}

// Root returns the MerkleRoot
func (mt *ZkTrieImpl) Root() *zkt.Hash {
	if mt.Debug {
		_, err := mt.GetNode(mt.rootKey)
		if err != nil {
			var hash common.Hash
			hash.SetBytes(mt.rootKey.Bytes())
			panic(fmt.Errorf("load trie root failed hash %v", hash))
		}
	}
	return mt.rootKey
}

// MaxLevels returns the MT maximum level
func (mt *ZkTrieImpl) MaxLevels() int {
	return mt.maxLevels
}

// Add adds a Key & Value into the ZkTrieImpl. Where the `k` determines the
// path from the Root to the Leaf.
func (mt *ZkTrieImpl) Add(k, v *big.Int, kPreimage *zkt.Byte32, vPreimage []byte) error {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k & v are valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return errors.New("Key not inside the Finite Field")
	}
	if !cryptoUtils.CheckBigIntInField(v) {
		return errors.New("Value not inside the Finite Field")
	}

	kHash := zkt.NewHashFromBigInt(k)
	vHash := zkt.NewHashFromBigInt(v)
	newNodeLeaf := NewNodeLeaf(kHash, vHash, kPreimage, vPreimage)
	path := getPath(mt.maxLevels, kHash[:])

	newRootKey, err := mt.addLeaf(newNodeLeaf, mt.rootKey, 0, path)
	if err == ErrEntryIndexAlreadyExists {
		newRootKey, err = mt.updateNode(newNodeLeaf)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	mt.rootKey = newRootKey
	err = mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
	if err != nil {
		return err
	}

	return nil
}

// AddWord
// Add a Bytes32 kv to ZkTrieImpl
func (mt *ZkTrieImpl) AddWord(kPreimage, vPreimage *zkt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	v, err := vPreimage.Hash()
	if err != nil {
		return err
	}
	err = mt.Add(k, v, kPreimage, vPreimage[:])
	return err
}

// AddVarWord
// Add a Bytes32 k and a var-length value to ZkTrieImpl
// User Must provide the hash of vPreimage since there is no define for vPreimage not Bytes32
func (mt *ZkTrieImpl) AddVarWord(kPreimage *zkt.Byte32, vHash *big.Int, vPreimage []byte) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	err = mt.Add(k, vHash, kPreimage, vPreimage[:])
	return err
}

// pushLeaf recursively pushes an existing oldLeaf down until its path diverges
// from newLeaf, at which point both leafs are stored, all while updating the
// path.
func (mt *ZkTrieImpl) pushLeaf(newLeaf *Node, oldLeaf *Node, lvl int,
	pathNewLeaf []bool, pathOldLeaf []bool) (*zkt.Hash, error) {
	if lvl > mt.maxLevels-2 {
		return nil, ErrReachedMaxLevel
	}
	var newNodeMiddle *Node
	if pathNewLeaf[lvl] == pathOldLeaf[lvl] { // We need to go deeper!
		nextKey, err := mt.pushLeaf(newLeaf, oldLeaf, lvl+1, pathNewLeaf, pathOldLeaf)
		if err != nil {
			return nil, err
		}
		if pathNewLeaf[lvl] { // go right
			newNodeMiddle = NewNodeMiddle(&zkt.HashZero, nextKey)
		} else { // go left
			newNodeMiddle = NewNodeMiddle(nextKey, &zkt.HashZero)
		}
		return mt.addNode(newNodeMiddle)
	}
	oldLeafKey, err := oldLeaf.Key()
	if err != nil {
		return nil, err
	}
	newLeafKey, err := newLeaf.Key()
	if err != nil {
		return nil, err
	}

	if pathNewLeaf[lvl] {
		newNodeMiddle = NewNodeMiddle(oldLeafKey, newLeafKey)
	} else {
		newNodeMiddle = NewNodeMiddle(newLeafKey, oldLeafKey)
	}
	// We can add newLeaf now.  We don't need to add oldLeaf because it's
	// already in the tree.
	_, err = mt.addNode(newLeaf)
	if err != nil {
		return nil, err
	}
	return mt.addNode(newNodeMiddle)
}

// addLeaf recursively adds a newLeaf in the MT while updating the path.
func (mt *ZkTrieImpl) addLeaf(newLeaf *Node, key *zkt.Hash,
	lvl int, path []bool) (*zkt.Hash, error) {
	var err error
	var nextKey *zkt.Hash
	if lvl > mt.maxLevels-1 {
		return nil, ErrReachedMaxLevel
	}
	n, err := mt.GetNode(key)
	if err != nil {
		fmt.Printf("addLeaf:GetNode err %v key %v root %v level %v\n", err, key, mt.rootKey, lvl)
		fmt.Printf("root %v\n", mt.Root())
		return nil, err
	}
	switch n.Type {
	case NodeTypeEmpty:
		// We can add newLeaf now
		{
			r, e := mt.addNode(newLeaf)
			if e != nil {
				fmt.Println("err on NodeTypeEmpty mt.addNode ", e)
			}
			return r, e
		}
	case NodeTypeLeaf:
		nKey := n.Entry[0]
		// Check if leaf node found contains the leaf node we are
		// trying to add
		newLeafKey := newLeaf.Entry[0]
		if bytes.Equal(nKey[:], newLeafKey[:]) {
			if bytes.Equal(n.Entry[1][:], newLeaf.Entry[1][:]) {
				// do nothing, duplicate entry
				// FIXME more optimization may needed here
				k, err := n.Key()
				if err != nil {
					panic("wtf")
				}
				return k, nil
			} else {
				fmt.Printf("ErrEntryIndexAlreadyExists nKey %v newLeafKey %v n.Entry[1] %v newLeaf.Entry[1] %v\n",
					nKey, newLeafKey, n.Entry[1], newLeaf.Entry[1])
				return nil, ErrEntryIndexAlreadyExists
			}
		}
		pathOldLeaf := getPath(mt.maxLevels, nKey[:])
		// We need to push newLeaf down until its path diverges from
		// n's path
		return mt.pushLeaf(newLeaf, n, lvl, path, pathOldLeaf)
	case NodeTypeMiddle:
		// We need to go deeper, continue traversing the tree, left or
		// right depending on path
		var newNodeMiddle *Node
		if path[lvl] { // go right
			nextKey, err = mt.addLeaf(newLeaf, n.ChildR, lvl+1, path)
			newNodeMiddle = NewNodeMiddle(n.ChildL, nextKey)
		} else { // go left
			nextKey, err = mt.addLeaf(newLeaf, n.ChildL, lvl+1, path)
			newNodeMiddle = NewNodeMiddle(nextKey, n.ChildR)
		}
		if err != nil {
			fmt.Printf("addLeaf:GetNode err %v level %v\n", err, lvl)
			return nil, err
		}
		// Update the node to reflect the modified child
		return mt.addNode(newNodeMiddle)
	default:
		return nil, ErrInvalidNodeFound
	}
}

// addNode adds a node into the MT.  Empty nodes are not stored in the tree;
// they are all the same and assumed to always exist.
func (mt *ZkTrieImpl) addNode(n *Node) (*zkt.Hash, error) {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return nil, ErrNotWritable
	}
	if n.Type == NodeTypeEmpty {
		return n.Key()
	}
	k, err := n.Key()
	if err != nil {
		return nil, err
	}
	v := n.Value()
	// Check that the node key doesn't already exist
	oldV, err := mt.db.Get(k[:])
	if err == nil {
		if !bytes.Equal(oldV, v) {
			return nil, ErrNodeKeyAlreadyExists
		}
	} else if err == ErrNotFound {
		// expected, do nothing
		//return nil, ErrKeyNotFound
	}
	err = mt.db.Put(k[:], v)
	return k, err
}

// updateNode updates an existing node in the MT.  Empty nodes are not stored
// in the tree; they are all the same and assumed to always exist.
func (mt *ZkTrieImpl) updateNode(n *Node) (*zkt.Hash, error) {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return nil, ErrNotWritable
	}
	if n.Type == NodeTypeEmpty {
		return n.Key()
	}
	k, err := n.Key()
	if err != nil {
		return nil, err
	}
	v := n.Value()
	err = mt.db.Put(k[:], v)
	return k, err
}

// Get returns the value of the leaf for the given key
func (mt *ZkTrieImpl) Get(k *big.Int) (*big.Int, *big.Int, []*zkt.Hash, error) {
	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return nil, nil, nil, errors.New("Key not inside the Finite Field")
	}

	kHash := zkt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])

	nextKey := mt.rootKey
	siblings := []*zkt.Hash{}
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return nil, nil, nil, err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return big.NewInt(0), big.NewInt(0), siblings, ErrKeyNotFound
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				return n.Entry[0].BigInt(), n.Entry[1].BigInt(), siblings, nil
			}
			return n.Entry[0].BigInt(), n.Entry[1].BigInt(), siblings, ErrKeyNotFound
		case NodeTypeMiddle:
			if path[i] {
				nextKey = n.ChildR
				siblings = append(siblings, n.ChildL)
			} else {
				nextKey = n.ChildL
				siblings = append(siblings, n.ChildR)
			}
		default:
			return nil, nil, nil, ErrInvalidNodeFound
		}
	}

	return nil, nil, nil, ErrReachedMaxLevel
}

func (mt *ZkTrieImpl) GetLeafNode(k *big.Int) (*Node, error) {
	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return nil, errors.New("key not inside the Finite Field")
	}

	kHash := zkt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])
	nextKey := mt.rootKey
	var siblings []*zkt.Hash
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return nil, err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return NewNodeEmpty(), ErrKeyNotFound
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				return n, nil
			}
			return n, ErrKeyNotFound
		case NodeTypeMiddle:
			if path[i] {
				nextKey = n.ChildR
				siblings = append(siblings, n.ChildL)
			} else {
				nextKey = n.ChildL
				siblings = append(siblings, n.ChildR)
			}
		default:
			return nil, ErrInvalidNodeFound
		}
	}

	return nil, ErrReachedMaxLevel
}

func (mt *ZkTrieImpl) GetLeafNodeByWord(kPreimage *zkt.Byte32) (*Node, error) {
	k, err := kPreimage.Hash()
	if err != nil {
		return nil, err
	}
	return mt.GetLeafNode(k)
}

// Update function updates the value of a specified key in the ZkTrieImpl, and updates
// the path from the leaf to the Root with the new values,and returns the
// CircomProcessorProof.
func (mt *ZkTrieImpl) Update(k, v *big.Int, kPreimage *zkt.Byte32, vPreimage []byte) error {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k & are valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return errors.New("Key not inside the Finite Field")
	}
	if !cryptoUtils.CheckBigIntInField(v) {
		return errors.New("Key not inside the Finite Field")
	}

	kHash := zkt.NewHashFromBigInt(k)
	vHash := zkt.NewHashFromBigInt(v)
	path := getPath(mt.maxLevels, kHash[:])

	nextKey := mt.rootKey
	siblings := []*zkt.Hash{}
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return ErrKeyNotFound
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				// update leaf and upload to the root
				newNodeLeaf := NewNodeLeaf(kHash, vHash, kPreimage, vPreimage)
				_, err := mt.updateNode(newNodeLeaf)
				if err != nil {
					return err
				}
				newRootKey, err :=
					mt.recalculatePathUntilRoot(path, newNodeLeaf, siblings)
				if err != nil {
					return err
				}
				mt.rootKey = newRootKey
				err = mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
				if err != nil {
					return err
				}
				return nil
			}
			return ErrKeyNotFound
		case NodeTypeMiddle:
			if path[i] {
				nextKey = n.ChildR
				siblings = append(siblings, n.ChildL)
			} else {
				nextKey = n.ChildL
				siblings = append(siblings, n.ChildR)
			}
		default:
			return ErrInvalidNodeFound
		}
	}

	return ErrKeyNotFound
}

func (mt *ZkTrieImpl) UpdateWord(kPreimage, vPreimage *zkt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	v, err := vPreimage.Hash()
	if err != nil {
		return err
	}
	err = mt.Update(k, v, kPreimage, vPreimage[:])
	if err == ErrKeyNotFound {
		err = mt.Add(k, v, kPreimage, vPreimage[:])
		return err
	} else if err != nil {
		fmt.Printf("UpdateWord err %v", err)
	}
	return err
}

func (mt *ZkTrieImpl) UpdateVarWord(kPreimage *zkt.Byte32, vHash *big.Int, vPreimage []byte) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	err = mt.Update(k, vHash, kPreimage, vPreimage[:])
	if err == ErrKeyNotFound {
		err = mt.Add(k, vHash, kPreimage, vPreimage[:])
		if err != nil {
			log.Error("UpdateVarWord, inset still failed %v root %v", err, mt.rootKey)
		}
		return err
	} else if err != nil {
		log.Error("UpdateVarWord err %v %v", err, reflect.TypeOf(err))
	}
	return err
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
func (mt *ZkTrieImpl) Delete(k *big.Int) error {
	// verify that the ZkTrieImpl is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return errors.New("Key not inside the Finite Field")
	}

	kHash := zkt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])

	nextKey := mt.rootKey
	siblings := []*zkt.Hash{}
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return ErrKeyNotFound
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				// remove and go up with the sibling
				err = mt.rmAndUpload(path, kHash, siblings)
				return err
			}
			return ErrKeyNotFound
		case NodeTypeMiddle:
			if path[i] {
				nextKey = n.ChildR
				siblings = append(siblings, n.ChildL)
			} else {
				nextKey = n.ChildL
				siblings = append(siblings, n.ChildR)
			}
		default:
			return ErrInvalidNodeFound
		}
	}

	return ErrKeyNotFound
}

func (mt *ZkTrieImpl) DeleteWord(kPreimage *zkt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	return mt.Delete(k)
}

// rmAndUpload removes the key, and goes up until the root updating all the
// nodes with the new values.
func (mt *ZkTrieImpl) rmAndUpload(path []bool, kHash *zkt.Hash, siblings []*zkt.Hash) error {
	if len(siblings) == 0 {
		mt.rootKey = &zkt.HashZero
		err := mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
		if err != nil {
			return err
		}
		return nil
	}

	toUpload := siblings[len(siblings)-1]
	if len(siblings) < 2 { //nolint:gomnd
		mt.rootKey = siblings[0]
		err := mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
		if err != nil {
			return err
		}
		return nil
	}
	for i := len(siblings) - 2; i >= 0; i-- { //nolint:gomnd
		if !bytes.Equal(siblings[i][:], zkt.HashZero[:]) {
			var newNode *Node
			if path[i] {
				newNode = NewNodeMiddle(siblings[i], toUpload)
			} else {
				newNode = NewNodeMiddle(toUpload, siblings[i])
			}
			_, err := mt.addNode(newNode)
			if err != ErrNodeKeyAlreadyExists && err != nil {
				return err
			}
			// go up until the root
			newRootKey, err := mt.recalculatePathUntilRoot(path, newNode,
				siblings[:i])
			if err != nil {
				return err
			}
			mt.rootKey = newRootKey
			err = mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
			if err != nil {
				return err
			}
			break
		}
		// if i==0 (root position), stop and store the sibling of the
		// deleted leaf as root
		if i == 0 {
			mt.rootKey = toUpload
			err := mt.dbInsert(dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

// recalculatePathUntilRoot recalculates the nodes until the Root
func (mt *ZkTrieImpl) recalculatePathUntilRoot(path []bool, node *Node,
	siblings []*zkt.Hash) (*zkt.Hash, error) {
	for i := len(siblings) - 1; i >= 0; i-- {
		nodeKey, err := node.Key()
		if err != nil {
			return nil, err
		}
		if path[i] {
			node = NewNodeMiddle(siblings[i], nodeKey)
		} else {
			node = NewNodeMiddle(nodeKey, siblings[i])
		}
		_, err = mt.addNode(node)
		if err != ErrNodeKeyAlreadyExists && err != nil {
			return nil, err
		}
	}

	// return last node added, which is the root
	nodeKey, err := node.Key()
	return nodeKey, err
}

// dbInsert is a helper function to insert a node into a key in an open db
// transaction.
func (mt *ZkTrieImpl) dbInsert(k []byte, t NodeType, data []byte) error {
	v := append([]byte{byte(t)}, data...)
	return mt.db.Put(k, v)
}

// GetNode gets a node by key from the MT.  Empty nodes are not stored in the
// tree; they are all the same and assumed to always exist.
// <del>for non exist key, return (NewNodeEmpty(), nil)</del>
func (mt *ZkTrieImpl) GetNode(key *zkt.Hash) (*Node, error) {
	if bytes.Equal(key[:], zkt.HashZero[:]) {
		return NewNodeEmpty(), nil
	}
	nBytes, err := mt.db.Get(key[:])
	if err == ErrNotFound {
		//return NewNodeEmpty(), nil
		return nil, ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}
	return NewNodeFromBytes(nBytes)
}

// getPath returns the binary path, from the root to the leaf.
func getPath(numLevels int, k []byte) []bool {
	path := make([]bool, numLevels)
	for n := 0; n < numLevels; n++ {
		path[n] = zkt.TestBit(k[:], uint(n))
	}
	return path
}

// NodeAux contains the auxiliary node used in a non-existence proof.
type NodeAux struct {
	Key   *zkt.Hash
	Value *zkt.Hash
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
	notempties [zkt.ElemBytesLen - proofFlagsLen]byte
	// Siblings is a list of non-empty sibling keys.
	Siblings []*zkt.Hash
	NodeAux  *NodeAux
}

// VerifyProof verifies the Merkle Proof for the entry and root.
func VerifyProofZkTrie(rootKey *zkt.Hash, proof *Proof, k, v *big.Int) bool {
	rootFromProof, err := RootFromProof(proof, k, v)
	if err != nil {
		return false
	}
	return bytes.Equal(rootKey[:], rootFromProof[:])
}

// RootFromProof calculates the root that would correspond to a tree whose
// siblings are the ones in the proof with the leaf hashing to hIndex and
// hValue.
func RootFromProof(proof *Proof, k, v *big.Int) (*zkt.Hash, error) {
	kHash := zkt.NewHashFromBigInt(k)
	vHash := zkt.NewHashFromBigInt(v)
	sibIdx := len(proof.Siblings) - 1
	var err error
	var midKey *zkt.Hash
	if proof.Existence {
		midKey, err = LeafKey(kHash, vHash)
		if err != nil {
			return nil, err
		}
	} else {
		if proof.NodeAux == nil {
			midKey = &zkt.HashZero
		} else {
			if bytes.Equal(kHash[:], proof.NodeAux.Key[:]) {
				return nil,
					fmt.Errorf("Non-existence proof being checked against hIndex equal to nodeAux")
			}
			midKey, err = LeafKey(proof.NodeAux.Key, proof.NodeAux.Value)
			if err != nil {
				return nil, err
			}
		}
	}
	path := getPath(int(proof.depth), kHash[:])
	var siblingKey *zkt.Hash
	for lvl := int(proof.depth) - 1; lvl >= 0; lvl-- {
		if zkt.TestBitBigEndian(proof.notempties[:], uint(lvl)) {
			siblingKey = proof.Siblings[sibIdx]
			sibIdx--
		} else {
			siblingKey = &zkt.HashZero
		}
		if path[lvl] {
			midKey, err = NewNodeMiddle(siblingKey, midKey).Key()
			if err != nil {
				return nil, err
			}
		} else {
			midKey, err = NewNodeMiddle(midKey, siblingKey).Key()
			if err != nil {
				return nil, err
			}
		}
	}
	return midKey, nil
}

// walk is a helper recursive function to iterate over all tree branches
func (mt *ZkTrieImpl) walk(key *zkt.Hash, f func(*Node)) error {
	n, err := mt.GetNode(key)
	if err != nil {
		return err
	}
	switch n.Type {
	case NodeTypeEmpty:
		f(n)
	case NodeTypeLeaf:
		f(n)
	case NodeTypeMiddle:
		f(n)
		if err := mt.walk(n.ChildL, f); err != nil {
			return err
		}
		if err := mt.walk(n.ChildR, f); err != nil {
			return err
		}
	default:
		return ErrInvalidNodeFound
	}
	return nil
}

// Walk iterates over all the branches of a ZkTrieImpl with the given rootKey
// if rootKey is nil, it will get the current RootKey of the current state of
// the ZkTrieImpl.  For each node, it calls the f function given in the
// parameters.  See some examples of the Walk function usage in the
// ZkTrieImpl.go and merkletree_test.go
func (mt *ZkTrieImpl) Walk(rootKey *zkt.Hash, f func(*Node)) error {
	if rootKey == nil {
		rootKey = mt.Root()
	}
	err := mt.walk(rootKey, f)
	return err
}

// GraphViz uses Walk function to generate a string GraphViz representation of
// the tree and writes it to w
func (mt *ZkTrieImpl) GraphViz(w io.Writer, rootKey *zkt.Hash) error {
	fmt.Fprintf(w, `digraph hierarchy {
node [fontname=Monospace,fontsize=10,shape=box]
`)
	cnt := 0
	var errIn error
	err := mt.Walk(rootKey, func(n *Node) {
		k, err := n.Key()
		if err != nil {
			errIn = err
		}
		switch n.Type {
		case NodeTypeEmpty:
		case NodeTypeLeaf:
			fmt.Fprintf(w, "\"%v\" [style=filled];\n", k.String())
		case NodeTypeMiddle:
			lr := [2]string{n.ChildL.String(), n.ChildR.String()}
			emptyNodes := ""
			for i := range lr {
				if lr[i] == "0" {
					lr[i] = fmt.Sprintf("empty%v", cnt)
					emptyNodes += fmt.Sprintf("\"%v\" [style=dashed,label=0];\n", lr[i])
					cnt++
				}
			}
			fmt.Fprintf(w, "\"%v\" -> {\"%v\" \"%v\"}\n", k.String(), lr[0], lr[1])
			fmt.Fprint(w, emptyNodes)
		default:
		}
	})
	fmt.Fprintf(w, "}\n")
	if errIn != nil {
		return errIn
	}
	return err
}

// PrintGraphViz prints directly the GraphViz() output
func (mt *ZkTrieImpl) PrintGraphViz(rootKey *zkt.Hash) error {
	if rootKey == nil {
		rootKey = mt.Root()
	}
	w := bytes.NewBufferString("")
	fmt.Fprintf(w,
		"--------\nGraphViz of the ZkTrieImpl with RootKey "+rootKey.BigInt().String()+"\n")
	err := mt.GraphViz(w, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(w,
		"End of GraphViz of the ZkTrieImpl with RootKey "+rootKey.BigInt().String()+"\n--------\n")

	fmt.Println(w)
	return nil
}
