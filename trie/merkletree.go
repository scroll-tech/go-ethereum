package trie

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types/smt"
	"github.com/ethereum/go-ethereum/trie/db"
	"io"
	"math/big"
	"sync"

	cryptoUtils "github.com/iden3/go-iden3-crypto/utils"
)

const (
	// proofFlagsLen is the byte length of the flags in the proof header
	// (first 32 bytes).
	proofFlagsLen = 2
)

var (
	// ErrNodeKeyAlreadyExists is used when a node key already exists.
	ErrNodeKeyAlreadyExists = errors.New("key already exists")
	// ErrKeyNotFound is used when a key is not found in the MerkleTree.
	ErrKeyNotFound = errors.New("Key not found in the MerkleTree")
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
	// ErrInvalidDBValue is used when a value in the key value DB is
	// invalid (for example, it doesn't contain a byte header and a []byte
	// body of at least len=1).
	ErrInvalidDBValue = errors.New("the value in the DB is invalid")
	// ErrEntryIndexAlreadyExists is used when the entry index already
	// exists in the tree.
	ErrEntryIndexAlreadyExists = errors.New("the entry index already exists in the tree")
	// ErrNotWritable is used when the MerkleTree is not writable and a
	// write function is called
	ErrNotWritable = errors.New("Merkle Tree not writable")

	dbKeyRootNode = []byte("currentroot")
	// HashZero is used at Empty nodes
)

// MerkleTree is the struct with the main elements of the MerkleTree
type MerkleTree struct {
	sync.RWMutex
	db        db.Storage
	rootKey   *smt.Hash
	writable  bool
	maxLevels int
}

func NewMerkleTree(storage db.Storage, maxLevels int) (*MerkleTree, error) {
	return NewMerkleTreeWithRoot(storage, &smt.HashZero, maxLevels)
}

// NewMerkleTreeWithRoot loads a new Merkletree. If in the storage already exists one
// will open that one, if not, will create a new one.
func NewMerkleTreeWithRoot(storage db.Storage, root *smt.Hash, maxLevels int) (*MerkleTree, error) {
	mt := MerkleTree{db: storage, maxLevels: maxLevels, writable: true}
	/*
		root, err := mt.dbGetRoot()
		if err == db.ErrNotFound {
			tx, err := mt.db.NewTx()
			if err != nil {
				return nil, err
			}
			mt.rootKey = &HashZero
			err = tx.Put(dbKeyRootNode, mt.rootKey[:])
			if err != nil {
				return nil, err
			}
			err = tx.Commit()
			if err != nil {
				return nil, err
			}
			return &mt, nil
		} else if err != nil {
			return nil, err
		}
	*/
	mt.rootKey = root
	return &mt, nil
}

// we give the root from external, like state root in block header
/*
func (mt *MerkleTree) dbGetRoot() (*Hash, error) {
	v, err := mt.db.Get(dbKeyRootNode)
	if err != nil {
		return nil, err
	}
	var root Hash
	// Skip first byte which contains the NodeType
	copy(root[:], v[1:])
	return &root, nil
}
*/

// DB returns the MerkleTree.DB()
func (mt *MerkleTree) DB() db.Storage {
	return mt.db
}

// Root returns the MerkleRoot
func (mt *MerkleTree) Root() *smt.Hash {
	return mt.rootKey
}

// MaxLevels returns the MT maximum level
func (mt *MerkleTree) MaxLevels() int {
	return mt.maxLevels
}

// Snapshot returns a read-only copy of the MerkleTree
func (mt *MerkleTree) Snapshot(rootKey *smt.Hash) (*MerkleTree, error) {
	mt.RLock()
	defer mt.RUnlock()
	_, err := mt.GetNode(rootKey)
	if err != nil {
		return nil, err
	}
	return &MerkleTree{db: mt.db, maxLevels: mt.maxLevels, rootKey: rootKey, writable: false}, nil
}

// Add adds a Key & Value into the MerkleTree. Where the `k` determines the
// path from the Root to the Leaf.
func (mt *MerkleTree) Add(k, v *big.Int, kPreimage *smt.Byte32, vPreimage []byte) error {
	// verify that the MerkleTree is writable
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

	tx, err := mt.db.NewTx()
	if err != nil {
		return err
	}
	mt.Lock()
	defer mt.Unlock()

	kHash := smt.NewHashFromBigInt(k)
	vHash := smt.NewHashFromBigInt(v)
	newNodeLeaf := NewNodeLeaf(kHash, vHash, kPreimage, vPreimage)
	path := getPath(mt.maxLevels, kHash[:])

	newRootKey, err := mt.addLeaf(tx, newNodeLeaf, mt.rootKey, 0, path)
	if err == ErrEntryIndexAlreadyExists {
		newRootKey, err = mt.updateNode(tx, newNodeLeaf)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	mt.rootKey = newRootKey
	err = mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// AddWord
// Add a Bytes32 kv to MerkleTree
func (mt *MerkleTree) AddWord(kPreimage, vPreimage *smt.Byte32) error {
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
// Add a Bytes32 k and a var-length value to MerkleTree
// User Must provide the hash of vPreimage since there is no define for vPreimage not Bytes32
func (mt *MerkleTree) AddVarWord(kPreimage *smt.Byte32, vHash *big.Int, vPreimage []byte) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	err = mt.Add(k, vHash, kPreimage, vPreimage[:])
	return err
}

// This function has no sense now.. just used for testing...
// AddAndGetCircomProof does an Add, and returns a CircomProcessorProof
func (mt *MerkleTree) AddAndGetCircomProof(k,
	v *big.Int) (*CircomProcessorProof, error) {
	var cp CircomProcessorProof
	cp.Fnc = 2
	cp.OldRoot = mt.rootKey
	gettedK, gettedV, _, err := mt.Get(k)
	if err != nil && err != ErrKeyNotFound {
		return nil, err
	}
	cp.OldKey = smt.NewHashFromBigInt(gettedK)
	cp.OldValue = smt.NewHashFromBigInt(gettedV)
	if bytes.Equal(cp.OldKey[:], smt.HashZero[:]) {
		cp.IsOld0 = true
	}
	_, _, siblings, err := mt.Get(k)
	if err != nil && err != ErrKeyNotFound {
		return nil, err
	}
	cp.Siblings = CircomSiblingsFromSiblings(siblings, mt.maxLevels)
	//cp.Siblings = siblings

	err = mt.Add(k, v, &byte32Zero, byte32Zero[:])
	if err != nil {
		return nil, err
	}

	cp.NewKey = smt.NewHashFromBigInt(k)
	cp.NewValue = smt.NewHashFromBigInt(v)
	cp.NewRoot = mt.rootKey

	return &cp, nil
}

// pushLeaf recursively pushes an existing oldLeaf down until its path diverges
// from newLeaf, at which point both leafs are stored, all while updating the
// path.
func (mt *MerkleTree) pushLeaf(tx db.Tx, newLeaf *Node, oldLeaf *Node, lvl int,
	pathNewLeaf []bool, pathOldLeaf []bool) (*smt.Hash, error) {
	if lvl > mt.maxLevels-2 {
		return nil, ErrReachedMaxLevel
	}
	var newNodeMiddle *Node
	if pathNewLeaf[lvl] == pathOldLeaf[lvl] { // We need to go deeper!
		nextKey, err := mt.pushLeaf(tx, newLeaf, oldLeaf, lvl+1, pathNewLeaf, pathOldLeaf)
		if err != nil {
			return nil, err
		}
		if pathNewLeaf[lvl] { // go right
			newNodeMiddle = NewNodeMiddle(&smt.HashZero, nextKey)
		} else { // go left
			newNodeMiddle = NewNodeMiddle(nextKey, &smt.HashZero)
		}
		return mt.addNode(tx, newNodeMiddle)
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
	_, err = mt.addNode(tx, newLeaf)
	if err != nil {
		return nil, err
	}
	return mt.addNode(tx, newNodeMiddle)
}

// addLeaf recursively adds a newLeaf in the MT while updating the path.
func (mt *MerkleTree) addLeaf(tx db.Tx, newLeaf *Node, key *smt.Hash,
	lvl int, path []bool) (*smt.Hash, error) {
	var err error
	var nextKey *smt.Hash
	if lvl > mt.maxLevels-1 {
		return nil, ErrReachedMaxLevel
	}
	n, err := mt.GetNode(key)
	if err != nil {
		return nil, err
	}
	switch n.Type {
	case NodeTypeEmpty:
		// We can add newLeaf now
		{
			r, e := mt.addNode(tx, newLeaf)
			if e != nil {
				fmt.Println("err on NodeTypeEmpty mt.addNode ")
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
		return mt.pushLeaf(tx, newLeaf, n, lvl, path, pathOldLeaf)
	case NodeTypeMiddle:
		// We need to go deeper, continue traversing the tree, left or
		// right depending on path
		var newNodeMiddle *Node
		if path[lvl] { // go right
			nextKey, err = mt.addLeaf(tx, newLeaf, n.ChildR, lvl+1, path)
			newNodeMiddle = NewNodeMiddle(n.ChildL, nextKey)
		} else { // go left
			nextKey, err = mt.addLeaf(tx, newLeaf, n.ChildL, lvl+1, path)
			newNodeMiddle = NewNodeMiddle(nextKey, n.ChildR)
		}
		if err != nil {
			return nil, err
		}
		// Update the node to reflect the modified child
		return mt.addNode(tx, newNodeMiddle)
	default:
		return nil, ErrInvalidNodeFound
	}
}

// addNode adds a node into the MT.  Empty nodes are not stored in the tree;
// they are all the same and assumed to always exist.
func (mt *MerkleTree) addNode(tx db.Tx, n *Node) (*smt.Hash, error) {
	// verify that the MerkleTree is writable
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
	oldV, err := tx.Get(k[:])
	if err == nil {
		if !bytes.Equal(oldV, v) {
			return nil, ErrNodeKeyAlreadyExists
		}
	}
	err = tx.Put(k[:], v)
	return k, err
}

// updateNode updates an existing node in the MT.  Empty nodes are not stored
// in the tree; they are all the same and assumed to always exist.
func (mt *MerkleTree) updateNode(tx db.Tx, n *Node) (*smt.Hash, error) {
	// verify that the MerkleTree is writable
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
	err = tx.Put(k[:], v)
	return k, err
}

// Get returns the value of the leaf for the given key
func (mt *MerkleTree) Get(k *big.Int) (*big.Int, *big.Int, []*smt.Hash, error) {
	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return nil, nil, nil, errors.New("Key not inside the Finite Field")
	}

	kHash := smt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])

	nextKey := mt.rootKey
	siblings := []*smt.Hash{}
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

func (mt *MerkleTree) GetLeafNode(k *big.Int) (*Node, error) {
	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return nil, errors.New("key not inside the Finite Field")
	}

	kHash := smt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])
	nextKey := mt.rootKey
	var siblings []*smt.Hash
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

func (mt *MerkleTree) GetLeafNodeByWord(kPreimage *smt.Byte32) (*Node, error) {
	k, err := kPreimage.Hash()
	if err != nil {
		return nil, err
	}
	return mt.GetLeafNode(k)
}

// Update function updates the value of a specified key in the MerkleTree, and updates
// the path from the leaf to the Root with the new values,and returns the
// CircomProcessorProof.
func (mt *MerkleTree) Update(k, v *big.Int, kPreimage *smt.Byte32, vPreimage []byte) (*CircomProcessorProof, error) {
	// verify that the MerkleTree is writable
	if !mt.writable {
		return nil, ErrNotWritable
	}

	// verify that k & are valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return nil, errors.New("Key not inside the Finite Field")
	}
	if !cryptoUtils.CheckBigIntInField(v) {
		return nil, errors.New("Key not inside the Finite Field")
	}
	tx, err := mt.db.NewTx()
	if err != nil {
		return nil, err
	}
	mt.Lock()
	defer mt.Unlock()

	kHash := smt.NewHashFromBigInt(k)
	vHash := smt.NewHashFromBigInt(v)
	path := getPath(mt.maxLevels, kHash[:])

	var cp CircomProcessorProof
	cp.Fnc = 1
	cp.OldRoot = mt.rootKey
	cp.OldKey = kHash
	cp.NewKey = kHash
	cp.NewValue = vHash

	nextKey := mt.rootKey
	siblings := []*smt.Hash{}
	for i := 0; i < mt.maxLevels; i++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return nil, err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return nil, ErrKeyNotFound
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				cp.OldValue = n.Entry[1]
				cp.Siblings = CircomSiblingsFromSiblings(siblings, mt.maxLevels)
				// update leaf and upload to the root
				newNodeLeaf := NewNodeLeaf(kHash, vHash, kPreimage, vPreimage)
				_, err := mt.updateNode(tx, newNodeLeaf)
				if err != nil {
					return nil, err
				}
				newRootKey, err :=
					mt.recalculatePathUntilRoot(tx, path, newNodeLeaf, siblings)
				if err != nil {
					return nil, err
				}
				mt.rootKey = newRootKey
				err = mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
				if err != nil {
					return nil, err
				}
				cp.NewRoot = newRootKey
				if err := tx.Commit(); err != nil {
					return nil, err
				}
				return &cp, nil
			}
			return nil, ErrKeyNotFound
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

	return nil, ErrKeyNotFound
}

func (mt *MerkleTree) UpdateWord(kPreimage, vPreimage *smt.Byte32) (*CircomProcessorProof, error) {
	k, err := kPreimage.Hash()
	if err != nil {
		return nil, err
	}
	v, err := vPreimage.Hash()
	if err != nil {
		return nil, err
	}
	proof, err := mt.Update(k, v, kPreimage, vPreimage[:])
	if err == ErrKeyNotFound {
		err = mt.Add(k, v, kPreimage, vPreimage[:])
		return nil, err
	}
	return proof, err
}

func (mt *MerkleTree) UpdateVarWord(kPreimage *smt.Byte32, vHash *big.Int, vPreimage []byte) (*CircomProcessorProof, error) {
	k, err := kPreimage.Hash()
	if err != nil {
		return nil, err
	}
	proof, err := mt.Update(k, vHash, kPreimage, vPreimage[:])
	if err == ErrKeyNotFound {
		err = mt.Add(k, vHash, kPreimage, vPreimage[:])
		return nil, err
	}
	return proof, err
}

// Delete removes the specified Key from the MerkleTree and updates the path
// from the deleted key to the Root with the new values.  This method removes
// the key from the MerkleTree, but does not remove the old nodes from the
// key-value database; this means that if the tree is accessed by an old Root
// where the key was not deleted yet, the key will still exist. If is desired
// to remove the key-values from the database that are not under the current
// Root, an option could be to dump all the leafs (using mt.DumpLeafs) and
// import them in a new MerkleTree in a new database (using
// mt.ImportDumpedLeafs), but this will lose all the Root history of the
// MerkleTree
func (mt *MerkleTree) Delete(k *big.Int) error {
	// verify that the MerkleTree is writable
	if !mt.writable {
		return ErrNotWritable
	}

	// verify that k is valid and fit inside the Finite Field.
	if !cryptoUtils.CheckBigIntInField(k) {
		return errors.New("Key not inside the Finite Field")
	}
	tx, err := mt.db.NewTx()
	if err != nil {
		return err
	}
	mt.Lock()
	defer mt.Unlock()

	kHash := smt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])

	nextKey := mt.rootKey
	siblings := []*smt.Hash{}
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
				err = mt.rmAndUpload(tx, path, kHash, siblings)
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

func (mt *MerkleTree) DeleteWord(kPreimage *smt.Byte32) error {
	k, err := kPreimage.Hash()
	if err != nil {
		return err
	}
	return mt.Delete(k)
}

// rmAndUpload removes the key, and goes up until the root updating all the
// nodes with the new values.
func (mt *MerkleTree) rmAndUpload(tx db.Tx, path []bool, kHash *smt.Hash, siblings []*smt.Hash) error {
	if len(siblings) == 0 {
		mt.rootKey = &smt.HashZero
		err := mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
		if err != nil {
			return err
		}
		return tx.Commit()
	}

	toUpload := siblings[len(siblings)-1]
	if len(siblings) < 2 { //nolint:gomnd
		mt.rootKey = siblings[0]
		err := mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
		if err != nil {
			return err
		}
		return tx.Commit()
	}
	for i := len(siblings) - 2; i >= 0; i-- { //nolint:gomnd
		if !bytes.Equal(siblings[i][:], smt.HashZero[:]) {
			var newNode *Node
			if path[i] {
				newNode = NewNodeMiddle(siblings[i], toUpload)
			} else {
				newNode = NewNodeMiddle(toUpload, siblings[i])
			}
			_, err := mt.addNode(tx, newNode)
			if err != ErrNodeKeyAlreadyExists && err != nil {
				return err
			}
			// go up until the root
			newRootKey, err := mt.recalculatePathUntilRoot(tx, path, newNode,
				siblings[:i])
			if err != nil {
				return err
			}
			mt.rootKey = newRootKey
			err = mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
			if err != nil {
				return err
			}
			break
		}
		// if i==0 (root position), stop and store the sibling of the
		// deleted leaf as root
		if i == 0 {
			mt.rootKey = toUpload
			err := mt.dbInsert(tx, dbKeyRootNode, DBEntryTypeRoot, mt.rootKey[:])
			if err != nil {
				return err
			}
			break
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

// recalculatePathUntilRoot recalculates the nodes until the Root
func (mt *MerkleTree) recalculatePathUntilRoot(tx db.Tx, path []bool, node *Node,
	siblings []*smt.Hash) (*smt.Hash, error) {
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
		_, err = mt.addNode(tx, node)
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
func (mt *MerkleTree) dbInsert(tx db.Tx, k []byte, t NodeType, data []byte) error {
	v := append([]byte{byte(t)}, data...)
	return tx.Put(k, v)
}

// GetNode gets a node by key from the MT.  Empty nodes are not stored in the
// tree; they are all the same and assumed to always exist.
func (mt *MerkleTree) GetNode(key *smt.Hash) (*Node, error) {
	if bytes.Equal(key[:], smt.HashZero[:]) {
		return NewNodeEmpty(), nil
	}
	nBytes, err := mt.db.Get(key[:])
	if err != nil {
		return nil, err
	}
	return NewNodeFromBytes(nBytes)
}

// getPath returns the binary path, from the root to the leaf.
func getPath(numLevels int, k []byte) []bool {
	path := make([]bool, numLevels)
	for n := 0; n < numLevels; n++ {
		path[n] = smt.TestBit(k[:], uint(n))
	}
	return path
}

// NodeAux contains the auxiliary node used in a non-existence proof.
type NodeAux struct {
	Key   *smt.Hash
	Value *smt.Hash
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
	notempties [smt.ElemBytesLen - proofFlagsLen]byte
	// Siblings is a list of non-empty sibling keys.
	Siblings []*smt.Hash
	NodeAux  *NodeAux
}

// NewProofFromBytes parses a byte array into a Proof.
func NewProofFromBytes(bs []byte) (*Proof, error) {
	if len(bs) < smt.ElemBytesLen {
		return nil, ErrInvalidProofBytes
	}
	p := &Proof{}
	if (bs[0] & 0x01) == 0 {
		p.Existence = true
	}
	p.depth = uint(bs[1])
	copy(p.notempties[:], bs[proofFlagsLen:smt.ElemBytesLen])
	siblingBytes := bs[smt.ElemBytesLen:]
	sibIdx := 0
	for i := uint(0); i < p.depth; i++ {
		if smt.TestBitBigEndian(p.notempties[:], i) {
			if len(siblingBytes) < (sibIdx+1)*smt.ElemBytesLen {
				return nil, ErrInvalidProofBytes
			}
			var sib smt.Hash
			copy(sib[:], siblingBytes[sibIdx*smt.ElemBytesLen:(sibIdx+1)*smt.ElemBytesLen])
			p.Siblings = append(p.Siblings, &sib)
			sibIdx++
		}
	}

	if !p.Existence && ((bs[0] & 0x02) != 0) {
		p.NodeAux = &NodeAux{Key: &smt.Hash{}, Value: &smt.Hash{}}
		nodeAuxBytes := siblingBytes[len(p.Siblings)*smt.ElemBytesLen:]
		if len(nodeAuxBytes) != 2*smt.ElemBytesLen {
			return nil, ErrInvalidProofBytes
		}
		copy(p.NodeAux.Key[:], nodeAuxBytes[:smt.ElemBytesLen])
		copy(p.NodeAux.Value[:], nodeAuxBytes[smt.ElemBytesLen:2*smt.ElemBytesLen])
	}
	return p, nil
}

// Bytes serializes a Proof into a byte array.
func (p *Proof) Bytes() []byte {
	bsLen := proofFlagsLen + len(p.notempties) + smt.ElemBytesLen*len(p.Siblings)
	if p.NodeAux != nil {
		bsLen += 2 * smt.ElemBytesLen //nolint:gomnd
	}
	bs := make([]byte, bsLen)

	if !p.Existence {
		bs[0] |= 0x01
	}
	bs[1] = byte(p.depth)
	copy(bs[proofFlagsLen:len(p.notempties)+proofFlagsLen], p.notempties[:])
	siblingsBytes := bs[len(p.notempties)+proofFlagsLen:]
	for i, k := range p.Siblings {
		copy(siblingsBytes[i*smt.ElemBytesLen:(i+1)*smt.ElemBytesLen], k[:])
	}
	if p.NodeAux != nil {
		bs[0] |= 0x02
		copy(bs[len(bs)-2*smt.ElemBytesLen:], p.NodeAux.Key[:])
		copy(bs[len(bs)-1*smt.ElemBytesLen:], p.NodeAux.Value[:])
	}
	return bs
}

// SiblingsFromProof returns all the siblings of the proof.
func SiblingsFromProof(proof *Proof) []*smt.Hash {
	sibIdx := 0
	siblings := []*smt.Hash{}
	for lvl := 0; lvl < int(proof.depth); lvl++ {
		if smt.TestBitBigEndian(proof.notempties[:], uint(lvl)) {
			siblings = append(siblings, proof.Siblings[sibIdx])
			sibIdx++
		} else {
			siblings = append(siblings, &smt.HashZero)
		}
	}
	return siblings
}

// AllSiblings returns all the siblings of the proof.
func (p *Proof) AllSiblings() []*smt.Hash {
	return SiblingsFromProof(p)
}

// CircomSiblingsFromSiblings returns the full siblings compatible with circom
func CircomSiblingsFromSiblings(siblings []*smt.Hash, levels int) []*smt.Hash {
	// Add the rest of empty levels to the siblings
	for i := len(siblings); i < levels+1; i++ {
		siblings = append(siblings, &smt.HashZero)
	}
	return siblings
}

// CircomProcessorProof defines the ProcessorProof compatible with circom. Is
// the data of the proof between the transition from one state to another.
type CircomProcessorProof struct {
	OldRoot  *smt.Hash   `json:"oldRoot"`
	NewRoot  *smt.Hash   `json:"newRoot"`
	Siblings []*smt.Hash `json:"siblings"`
	OldKey   *smt.Hash   `json:"oldKey"`
	OldValue *smt.Hash   `json:"oldValue"`
	NewKey   *smt.Hash   `json:"newKey"`
	NewValue *smt.Hash   `json:"newValue"`
	IsOld0   bool        `json:"isOld0"`
	// 0: NOP, 1: Update, 2: Insert, 3: Delete
	Fnc int `json:"fnc"`
}

// String function returns a human-readable string representation of the
// CircomProcessorProof
func (p CircomProcessorProof) String() string {
	buf := bytes.NewBufferString("{")
	fmt.Fprintf(buf, "	OldRoot: %v,\n", p.OldRoot)
	fmt.Fprintf(buf, "	NewRoot: %v,\n", p.NewRoot)
	fmt.Fprintf(buf, "	Siblings: [\n		")
	for _, s := range p.Siblings {
		fmt.Fprintf(buf, "%v, ", s)
	}
	fmt.Fprintf(buf, "\n	],\n")
	fmt.Fprintf(buf, "	OldKey: %v,\n", p.OldKey)
	fmt.Fprintf(buf, "	OldValue: %v,\n", p.OldValue)
	fmt.Fprintf(buf, "	NewKey: %v,\n", p.NewKey)
	fmt.Fprintf(buf, "	NewValue: %v,\n", p.NewValue)
	fmt.Fprintf(buf, "	IsOld0: %v,\n", p.IsOld0)
	fmt.Fprintf(buf, "	Fnc: %v,\n", p.Fnc)
	fmt.Fprintf(buf, "}\n")

	return buf.String()
}

// CircomVerifierProof defines the VerifierProof compatible with circom. Is the
// data of the proof that a certain leaf exists in the MerkleTree.
type CircomVerifierProof struct {
	Root     *smt.Hash   `json:"root"`
	Siblings []*smt.Hash `json:"siblings"`
	OldKey   *smt.Hash   `json:"oldKey"`
	OldValue *smt.Hash   `json:"oldValue"`
	IsOld0   bool        `json:"isOld0"`
	Key      *smt.Hash   `json:"key"`
	Value    *smt.Hash   `json:"value"`
	Fnc      int         `json:"fnc"` // 0: inclusion, 1: non inclusion
}

// GenerateCircomVerifierProof returns the CircomVerifierProof for a certain
// key in the MerkleTree.  If the rootKey is nil, the current merkletree root
// is used.
func (mt *MerkleTree) GenerateCircomVerifierProof(k *big.Int,
	rootKey *smt.Hash) (*CircomVerifierProof, error) {
	cp, err := mt.GenerateSCVerifierProof(k, rootKey)
	if err != nil {
		return nil, err
	}
	cp.Siblings = CircomSiblingsFromSiblings(cp.Siblings, mt.maxLevels)
	return cp, nil
}

// GenerateSCVerifierProof returns the CircomVerifierProof for a certain key in
// the MerkleTree with the Siblings without the extra 0 needed at the circom
// circuits, which makes it straight forward to verify inside a Smart
// Contract. If the rootKey is nil, the current merkletree root is used.
func (mt *MerkleTree) GenerateSCVerifierProof(k *big.Int,
	rootKey *smt.Hash) (*CircomVerifierProof, error) {
	if rootKey == nil {
		rootKey = mt.Root()
	}
	p, v, err := mt.GenerateProof(k, rootKey)
	if err != nil && err != ErrKeyNotFound {
		return nil, err
	}
	var cp CircomVerifierProof
	cp.Root = rootKey
	cp.Siblings = p.AllSiblings()
	if p.NodeAux != nil {
		cp.OldKey = p.NodeAux.Key
		cp.OldValue = p.NodeAux.Value
	} else {
		cp.OldKey = &smt.HashZero
		cp.OldValue = &smt.HashZero
	}
	cp.Key = smt.NewHashFromBigInt(k)
	cp.Value = smt.NewHashFromBigInt(v)
	if p.Existence {
		cp.Fnc = 0 // inclusion
	} else {
		cp.Fnc = 1 // non inclusion
	}

	return &cp, nil
}

// GenerateProof generates the proof of existence (or non-existence) of an
// Entry's hash Index for a Merkle Tree given the root.
// If the rootKey is nil, the current merkletree root is used
func (mt *MerkleTree) GenerateProof(k *big.Int, rootKey *smt.Hash) (*Proof,
	*big.Int, error) {
	p := &Proof{}
	var siblingKey *smt.Hash

	kHash := smt.NewHashFromBigInt(k)
	path := getPath(mt.maxLevels, kHash[:])
	if rootKey == nil {
		rootKey = mt.Root()
	}
	nextKey := rootKey
	for p.depth = 0; p.depth < uint(mt.maxLevels); p.depth++ {
		n, err := mt.GetNode(nextKey)
		if err != nil {
			return nil, nil, err
		}
		switch n.Type {
		case NodeTypeEmpty:
			return p, big.NewInt(0), nil
		case NodeTypeLeaf:
			if bytes.Equal(kHash[:], n.Entry[0][:]) {
				p.Existence = true
				return p, n.Entry[1].BigInt(), nil
			}
			// We found a leaf whose entry didn't match hIndex
			p.NodeAux = &NodeAux{Key: n.Entry[0], Value: n.Entry[1]}
			return p, n.Entry[1].BigInt(), nil
		case NodeTypeMiddle:
			if path[p.depth] {
				nextKey = n.ChildR
				siblingKey = n.ChildL
			} else {
				nextKey = n.ChildL
				siblingKey = n.ChildR
			}
		default:
			return nil, nil, ErrInvalidNodeFound
		}
		if !bytes.Equal(siblingKey[:], smt.HashZero[:]) {
			smt.SetBitBigEndian(p.notempties[:], uint(p.depth))
			p.Siblings = append(p.Siblings, siblingKey)
		}
	}
	return nil, nil, ErrKeyNotFound
}

// VerifyProof verifies the Merkle Proof for the entry and root.
func VerifyProofSMT(rootKey *smt.Hash, proof *Proof, k, v *big.Int) bool {
	rootFromProof, err := RootFromProof(proof, k, v)
	if err != nil {
		return false
	}
	return bytes.Equal(rootKey[:], rootFromProof[:])
}

// RootFromProof calculates the root that would correspond to a tree whose
// siblings are the ones in the proof with the leaf hashing to hIndex and
// hValue.
func RootFromProof(proof *Proof, k, v *big.Int) (*smt.Hash, error) {
	kHash := smt.NewHashFromBigInt(k)
	vHash := smt.NewHashFromBigInt(v)
	sibIdx := len(proof.Siblings) - 1
	var err error
	var midKey *smt.Hash
	if proof.Existence {
		midKey, err = LeafKey(kHash, vHash)
		if err != nil {
			return nil, err
		}
	} else {
		if proof.NodeAux == nil {
			midKey = &smt.HashZero
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
	var siblingKey *smt.Hash
	for lvl := int(proof.depth) - 1; lvl >= 0; lvl-- {
		if smt.TestBitBigEndian(proof.notempties[:], uint(lvl)) {
			siblingKey = proof.Siblings[sibIdx]
			sibIdx--
		} else {
			siblingKey = &smt.HashZero
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
func (mt *MerkleTree) walk(key *smt.Hash, f func(*Node)) error {
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

// Walk iterates over all the branches of a MerkleTree with the given rootKey
// if rootKey is nil, it will get the current RootKey of the current state of
// the MerkleTree.  For each node, it calls the f function given in the
// parameters.  See some examples of the Walk function usage in the
// merkletree.go and merkletree_test.go
func (mt *MerkleTree) Walk(rootKey *smt.Hash, f func(*Node)) error {
	if rootKey == nil {
		rootKey = mt.Root()
	}
	err := mt.walk(rootKey, f)
	return err
}

// GraphViz uses Walk function to generate a string GraphViz representation of
// the tree and writes it to w
func (mt *MerkleTree) GraphViz(w io.Writer, rootKey *smt.Hash) error {
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
func (mt *MerkleTree) PrintGraphViz(rootKey *smt.Hash) error {
	if rootKey == nil {
		rootKey = mt.Root()
	}
	w := bytes.NewBufferString("")
	fmt.Fprintf(w,
		"--------\nGraphViz of the MerkleTree with RootKey "+rootKey.BigInt().String()+"\n")
	err := mt.GraphViz(w, nil)
	if err != nil {
		return err
	}
	fmt.Fprintf(w,
		"End of GraphViz of the MerkleTree with RootKey "+rootKey.BigInt().String()+"\n--------\n")

	fmt.Println(w)
	return nil
}

// DumpLeafs returns all the Leafs that exist under the given Root. If no Root
// is given (nil), it uses the current Root of the MerkleTree.
func (mt *MerkleTree) DumpLeafs(rootKey *smt.Hash) ([]byte, error) {
	var b []byte
	err := mt.Walk(rootKey, func(n *Node) {
		if n.Type == NodeTypeLeaf {
			l := n.Entry[0].Bytes()
			r := n.Entry[1].Bytes()
			b = append(b, append(l[:], r[:]...)...)
			fmt.Printf("leaf %v %v\n", n.Entry[0].BigInt(), n.Entry[1].BigInt())
		}
	})
	return b, err
}

// ImportDumpedLeafs parses and adds to the MerkleTree the dumped list of leafs
// from the DumpLeafs function.
//func (mt *MerkleTree) ImportDumpedLeafs(b []byte) error {
//	for i := 0; i < len(b); i += 64 {
//		lr := b[i : i+64]
//		lB, err := NewBigIntFromHashBytes(lr[:32])
//		if err != nil {
//			return err
//		}
//		rB, err := NewBigIntFromHashBytes(lr[32:])
//		if err != nil {
//			return err
//		}
//		err = mt.Add(lB, rB)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
