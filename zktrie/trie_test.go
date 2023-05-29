// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zktrie

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"testing/quick"

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/sha3"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/codehash"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/rlp"
)

func init() {
	spew.Config.Indent = "    "
	spew.Config.DisableMethods = false
}

// Used for testing
func newEmpty() *Trie {
	trie, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	return trie
}

// makeTestTrie create a sample test trie to test node-wise reconstruction.
func makeTestTrie(t *testing.T) (*Database, *Trie, map[string][]byte) {
	// Create an empty trie
	triedb := NewDatabase(memorydb.New())
	trie, _ := New(common.Hash{}, triedb)

	// Fill it with some arbitrary data
	content := make(map[string][]byte)
	for i := byte(0); i < 255; i++ {
		// Map the same data under multiple keys
		key, val := common.RightPadBytes([]byte{1, i}, 32), common.LeftPadBytes([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		key, val = common.RightPadBytes([]byte{2, i}, 32), common.LeftPadBytes([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		// Add some other data to inflate the trie
		for j := byte(3); j < 13; j++ {
			key, val = common.RightPadBytes([]byte{j, i}, 32), common.LeftPadBytes([]byte{j, i}, 32)
			content[string(key)] = val
			trie.Update(key, val)
		}
	}
	_, _, err := trie.Commit(nil)
	if err != nil {
		t.Error(err)
	}

	// Return the generated trie
	return triedb, trie, content
}

func TestEmptyTrie(t *testing.T) {
	var trie Trie
	res := trie.Hash()
	exp := emptyRoot
	if res != exp {
		t.Errorf("expected %x got %x", exp, res)
	}
}

func TestNull(t *testing.T) {
	t.Skip("zk-trie will only be accessed with correct construction.")
	var trie Trie
	key := make([]byte, 32)
	value := []byte("test")
	trie.Update(key, value)
	if !bytes.Equal(trie.Get(key), value) {
		t.Fatal("wrong value")
	}
}

func TestMissingRoot(t *testing.T) {
	trie, err := New(common.HexToHash("0beec7b5ea3f0fdbc95d0dd47f3c5bc275da8a33"), NewDatabase(memorydb.New()))
	if trie != nil {
		t.Error("New returned non-nil trie for invalid root")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("New returned wrong error: %v", err)
	}
}

func TestMissingNode(t *testing.T) {
	diskdb := memorydb.New()
	triedb := NewDatabase(diskdb)

	trie, _ := New(common.Hash{}, triedb)
	updateString(trie, "12000000000000000000000000000000", "qwerqwerqwerqwerqwerqwerqwerqwer")
	updateString(trie, "12345600000000000000000000000000", "asdfasdfasdfasdfasdfasdfasdfasdf")
	root, _, _ := trie.Commit(nil)
	triedb.Commit(root, true, nil)

	trie, _ = New(root, triedb)
	_, err := trie.TryGet([]byte("12000000000000000000000000000000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("12009900000000000000000000000000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	_, err = trie.TryGet([]byte("12345600000000000000000000000000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryUpdate([]byte("12009900000000000000000000000000"), []byte("zxcvzxcvzxcvzxcvzxcvzxcvzxcvzxcv"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	trie, _ = New(root, triedb)
	err = trie.TryDelete([]byte("12345600000000000000000000000000"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	rootHash, _ := diskdb.Get([]byte("currentroot"))
	diskdb.Delete(rootHash[1:])

	// zk trie will validate database on construction
	trie, err = New(root, triedb)
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("New returned wrong error: %v", err)
	}
}

func TestInsert(t *testing.T) {
	trie := newEmpty()

	updateString(trie, "doe00000000000000000000000000000", "reindeer")
	updateString(trie, "dog00000000000000000000000000000", "puppy")
	updateString(trie, "dogglesworth00000000000000000000", "cat")

	exp := common.HexToHash("19c4352b0146c60b62d17a2195ec4e0d73fc241c4c49f7c0213bfe81bebf3180")
	root := trie.Hash()
	if root != exp {
		t.Errorf("case 1: exp %x got %x", exp, root)
	}

	trie = newEmpty()
	updateString(trie, "A0000000000000000000000000000000", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")

	exp = common.HexToHash("02770c4fa404a639a009590050da26aa360930dddfa4c7d789f9bb6cdff5ef03")
	root, _, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if root != exp {
		t.Errorf("case 2: exp %x got %x", exp, root)
	}
}

func TestGet(t *testing.T) {
	trie := newEmpty()
	// zk-trie modifies pass-in value to be 32-byte long
	var value32bytes = "xxxxxxxxxxxxxxxxxxxxxxxxxxxpuppy"
	updateString(trie, "doe00000000000000000000000000000", "reindeer")
	updateString(trie, "dog00000000000000000000000000000", value32bytes)
	updateString(trie, "dogglesworth00000000000000000000", "cat")

	for i := 0; i < 2; i++ {
		res := getString(trie, "dog00000000000000000000000000000")
		if !bytes.Equal(res, []byte(value32bytes)) {
			t.Errorf("expected %x got %x", value32bytes, res)
		}

		unknown := getString(trie, "unknown")
		if unknown != nil {
			t.Errorf("expected nil got %x", unknown)
		}

		if i == 1 {
			return
		}
		trie.Commit(nil)
	}
}

func TestDelete(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do000000000000000000000000000000", "verb"},
		{"ether000000000000000000000000000", "wookiedoo"},
		{"horse000000000000000000000000000", "stallion"},
		{"shaman00000000000000000000000000", "horse"},
		{"doge0000000000000000000000000000", "coin"},
		{"ether000000000000000000000000000", ""},
		{"dog00000000000000000000000000000", "puppy"},
		{"shaman00000000000000000000000000", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			updateString(trie, val.k, val.v)
		} else {
			deleteString(trie, val.k)
		}
	}

	hash := trie.Hash()
	exp := common.HexToHash("135b24c8e837dc5fd30d53217fe92a24073435adec97e24dd58cb7f1b4a4044e")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestEmptyValues(t *testing.T) {
	trie := newEmpty()

	vals := []struct{ k, v string }{
		{"do000000000000000000000000000000", "verb"},
		{"ether000000000000000000000000000", "wookiedoo"},
		{"horse000000000000000000000000000", "stallion"},
		{"shaman00000000000000000000000000", "horse"},
		{"doge0000000000000000000000000000", "coin"},
		{"ether000000000000000000000000000", ""},
		{"dog00000000000000000000000000000", "puppy"},
		{"shaman00000000000000000000000000", ""},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}

	hash := trie.Hash()
	exp := common.HexToHash("1638289eef5e066f49744706057781b954c5d9ef9f19fa49e2c8df3b6adbfe87")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestReplication(t *testing.T) {
	trie := newEmpty()
	vals := []struct{ k, v string }{
		{"do000000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxverb"},
		{"ether000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxwookiedoo"},
		{"horse000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxstallion"},
		{"shaman00000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxhorse"},
		{"doge0000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxcoin"},
		{"dog00000000000000000000000000000", "xxxxxxxxxxxxxxxxxxxxxxxxxxxpuppy"},
		{"somethingveryoddindeedthis is000", "xxxxxxxxxxxxxxxxxmyothernodedata"},
	}
	for _, val := range vals {
		updateString(trie, val.k, val.v)
	}
	exp, _, err := trie.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}

	// create a new trie on top of the database and check that lookups work.
	trie2, err := New(exp, trie.db)
	if err != nil {
		t.Fatalf("can't recreate trie at %x: %v", exp, err)
	}
	for _, kv := range vals {
		if string(getString(trie2, kv.k)) != kv.v {
			t.Errorf("trie2 doesn't have %q => %q", kv.k, kv.v)
		}
	}
	hash, _, err := trie2.Commit(nil)
	if err != nil {
		t.Fatalf("commit error: %v", err)
	}
	if hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}

	// perform some insertions on the new trie.
	vals2 := []struct{ k, v string }{
		{"do", "xxxxxxxxxxxxxxxxxxxxxxxxxxxxverb"},
		{"ether", "xxxxxxxxxxxxxxxxxxxxxxxwookiedoo"},
		{"horse", "xxxxxxxxxxxxxxxxxxxxxxxxstallion"},
		// {"shaman", "horse"},
		// {"doge", "coin"},
		// {"ether", ""},
		// {"dog", "puppy"},
		// {"somethingveryoddindeedthis is", "myothernodedata"},
		// {"shaman", ""},
	}
	for _, val := range vals2 {
		updateString(trie2, val.k, val.v)
	}
	if hash := trie2.Hash(); hash != exp {
		t.Errorf("root failure. expected %x got %x", exp, hash)
	}
}

func TestLargeValue(t *testing.T) {
	trie := newEmpty()
	trie.Update([]byte("key1"), []byte{99, 99, 99, 99})
	trie.Update([]byte("key2"), bytes.Repeat([]byte{1}, 32))
	trie.Hash()
}

// TestRandomCases tests som cases that were found via random fuzzing
func TestRandomCases(t *testing.T) {
	//TODO(kevinyum): re-enable after iterator is implemented
	t.Skip("re-enable after zk-trie implements iterator")
	var rt = []randTestStep{
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 0
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 1
		{op: 0, key: common.Hex2Bytes("d51b182b95d677e5f1c82508c0228de96b73092d78ce78b2230cd948674f66fd1483bd"), value: common.Hex2Bytes("0000000000000002")},           // step 2
		{op: 2, key: common.Hex2Bytes("c2a38512b83107d665c65235b0250002882ac2022eb00711552354832c5f1d030d0e408e"), value: common.Hex2Bytes("")},                         // step 3
		{op: 3, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 4
		{op: 3, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 5
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 6
		{op: 3, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 7
		{op: 0, key: common.Hex2Bytes("c2a38512b83107d665c65235b0250002882ac2022eb00711552354832c5f1d030d0e408e"), value: common.Hex2Bytes("0000000000000008")},         // step 8
		{op: 0, key: common.Hex2Bytes("d51b182b95d677e5f1c82508c0228de96b73092d78ce78b2230cd948674f66fd1483bd"), value: common.Hex2Bytes("0000000000000009")},           // step 9
		{op: 2, key: common.Hex2Bytes("fd"), value: common.Hex2Bytes("")},                                                                                               // step 10
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 11
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 12
		{op: 0, key: common.Hex2Bytes("fd"), value: common.Hex2Bytes("000000000000000d")},                                                                               // step 13
		{op: 6, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 14
		{op: 1, key: common.Hex2Bytes("c2a38512b83107d665c65235b0250002882ac2022eb00711552354832c5f1d030d0e408e"), value: common.Hex2Bytes("")},                         // step 15
		{op: 3, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 16
		{op: 0, key: common.Hex2Bytes("c2a38512b83107d665c65235b0250002882ac2022eb00711552354832c5f1d030d0e408e"), value: common.Hex2Bytes("0000000000000011")},         // step 17
		{op: 5, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 18
		{op: 3, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 19
		{op: 0, key: common.Hex2Bytes("d51b182b95d677e5f1c82508c0228de96b73092d78ce78b2230cd948674f66fd1483bd"), value: common.Hex2Bytes("0000000000000014")},           // step 20
		{op: 0, key: common.Hex2Bytes("d51b182b95d677e5f1c82508c0228de96b73092d78ce78b2230cd948674f66fd1483bd"), value: common.Hex2Bytes("0000000000000015")},           // step 21
		{op: 0, key: common.Hex2Bytes("c2a38512b83107d665c65235b0250002882ac2022eb00711552354832c5f1d030d0e408e"), value: common.Hex2Bytes("0000000000000016")},         // step 22
		{op: 5, key: common.Hex2Bytes(""), value: common.Hex2Bytes("")},                                                                                                 // step 23
		{op: 1, key: common.Hex2Bytes("980c393656413a15c8da01978ed9f89feb80b502f58f2d640e3a2f5f7a99a7018f1b573befd92053ac6f78fca4a87268"), value: common.Hex2Bytes("")}, // step 24
		{op: 1, key: common.Hex2Bytes("fd"), value: common.Hex2Bytes("")},                                                                                               // step 25
	}
	runRandTest(rt)

}

// randTest performs random trie operations.
// Instances of this test are created by Generate.
type randTest []randTestStep

type randTestStep struct {
	op    int
	key   []byte // for opUpdate, opDelete, opGet
	value []byte // for opUpdate
	err   error  // for debugging
}

const (
	opUpdate = iota
	opDelete
	opGet
	opCommit
	opHash
	opReset
	opItercheckhash
	opMax // boundary value, not an actual op
)

func (randTest) Generate(r *rand.Rand, size int) reflect.Value {
	var allKeys [][]byte
	genKey := func() []byte {
		if len(allKeys) < 2 || r.Intn(100) < 10 {
			// new key
			key := make([]byte, r.Intn(50))
			r.Read(key)
			allKeys = append(allKeys, key)
			return key
		}
		// use existing key
		return allKeys[r.Intn(len(allKeys))]
	}

	var steps randTest
	for i := 0; i < size; i++ {
		step := randTestStep{op: r.Intn(opMax)}
		switch step.op {
		case opUpdate:
			step.key = genKey()
			step.value = make([]byte, 8)
			binary.BigEndian.PutUint64(step.value, uint64(i))
		case opGet, opDelete:
			step.key = genKey()
		}
		steps = append(steps, step)
	}
	return reflect.ValueOf(steps)
}

func runRandTest(rt randTest) bool {
	triedb := NewDatabase(memorydb.New())

	tr, _ := New(common.Hash{}, triedb)
	values := make(map[string]string) // tracks content of the trie

	for i, step := range rt {
		fmt.Printf("{op: %d, key: common.Hex2Bytes(\"%x\"), value: common.Hex2Bytes(\"%x\")}, // step %d\n",
			step.op, step.key, step.value, i)
		switch step.op {
		case opUpdate:
			tr.Update(step.key, step.value)
			values[string(step.key)] = string(step.value)
		case opDelete:
			tr.Delete(step.key)
			delete(values, string(step.key))
		case opGet:
			v := tr.Get(step.key)
			want := values[string(step.key)]
			if string(v) != want {
				rt[i].err = fmt.Errorf("mismatch for key 0x%x, got 0x%x want 0x%x", step.key, v, want)
			}
		case opCommit:
			_, _, rt[i].err = tr.Commit(nil)
		case opHash:
			tr.Hash()
		case opReset:
			hash, _, err := tr.Commit(nil)
			if err != nil {
				rt[i].err = err
				return false
			}
			newtr, err := New(hash, triedb)
			if err != nil {
				rt[i].err = err
				return false
			}
			tr = newtr
		case opItercheckhash:
			checktr, _ := New(common.Hash{}, triedb)
			it := NewIterator(tr.NodeIterator(nil))
			for it.Next() {
				checktr.Update(it.Key, it.Value)
			}
			if tr.Hash() != checktr.Hash() {
				rt[i].err = fmt.Errorf("hash mismatch in opItercheckhash")
			}
		}
		// Abort the test on error.
		if rt[i].err != nil {
			return false
		}
	}
	return true
}

func TestRandom(t *testing.T) {
	//TODO(kevinyum): re-enable after iterator is implemented
	t.Skip("re-enable after zk-trie implements iterator")
	if err := quick.Check(runRandTest, nil); err != nil {
		if cerr, ok := err.(*quick.CheckError); ok {
			t.Fatalf("random test iteration %d failed: %s", cerr.Count, spew.Sdump(cerr.In))
		}
		t.Fatal(err)
	}
}

func TestTinyTrie(t *testing.T) {
	//TODO(kevinyum): re-enable after iterator is implemented
	t.Skip("re-enable after zk-trie implements iterator")
	// Create a realistic account trie to hash
	_, accounts := makeAccounts(5)
	trie := newEmpty()
	trie.Update(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000001337"), accounts[3])
	if exp, root := common.HexToHash("fc516c51c03bf9f1a0eec6ed6f6f5da743c2745dcd5670007519e6ec056f95a8"), trie.Hash(); exp != root {
		t.Errorf("1: got %x, exp %x", root, exp)
	}
	trie.Update(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000001338"), accounts[4])
	if exp, root := common.HexToHash("5070d3f144546fd13589ad90cd153954643fa4ca6c1a5f08683cbfbbf76e960c"), trie.Hash(); exp != root {
		t.Errorf("2: got %x, exp %x", root, exp)
	}
	trie.Update(common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000001339"), accounts[4])
	if exp, root := common.HexToHash("aa3fba77e50f6e931d8aacde70912be5bff04c7862f518ae06f3418dd4d37be3"), trie.Hash(); exp != root {
		t.Errorf("3: got %x, exp %x", root, exp)
	}
	checktr, _ := New(common.Hash{}, trie.db)
	it := NewIterator(trie.NodeIterator(nil))
	for it.Next() {
		checktr.Update(it.Key, it.Value)
	}
	if troot, itroot := trie.Hash(), checktr.Hash(); troot != itroot {
		t.Fatalf("hash mismatch in opItercheckhash, trie: %x, check: %x", troot, itroot)
	}
}

func TestCommitAfterHash(t *testing.T) {
	// Create a realistic account trie to hash
	addresses, accounts := makeAccounts(1000)
	trie := newEmpty()
	for i := 0; i < len(addresses); i++ {
		trie.Update(crypto.Keccak256(addresses[i][:]), accounts[i])
	}
	// Insert the accounts into the trie and hash it
	trie.Hash()
	trie.Commit(nil)
	root := trie.Hash()
	exp := common.HexToHash("2ed0586dd148735d1345859e44f2961b8adf7c139c88dafe5f3e4eab556e93e8")
	if exp != root {
		t.Errorf("got %x, exp %x", root, exp)
	}
	root, _, _ = trie.Commit(nil)
	if exp != root {
		t.Errorf("got %x, exp %x", root, exp)
	}
}

func makeAccounts(size int) (addresses [][20]byte, accounts [][]byte) {
	// Make the random benchmark deterministic
	random := rand.New(rand.NewSource(0))
	// Create a realistic account trie to hash
	addresses = make([][20]byte, size)
	for i := 0; i < len(addresses); i++ {
		data := make([]byte, 20)
		random.Read(data)
		copy(addresses[i][:], data)
	}
	accounts = make([][]byte, len(addresses))
	for i := 0; i < len(accounts); i++ {
		var (
			nonce = uint64(random.Int63())
			root  = emptyRoot
		)
		// The big.Rand function is not deterministic with regards to 64 vs 32 bit systems,
		// and will consume different amount of data from the rand source.
		//balance = new(big.Int).Rand(random, new(big.Int).Exp(common.Big2, common.Big256, nil))
		// Therefore, we instead just read via byte buffer
		numBytes := random.Uint32() % 33 // [0, 32] bytes
		balanceBytes := make([]byte, numBytes)
		random.Read(balanceBytes)
		balance := new(big.Int).SetBytes(balanceBytes)

		data, _ := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:            nonce,
			Balance:          balance,
			Root:             root,
			KeccakCodeHash:   codehash.EmptyKeccakCodeHash.Bytes(),
			PoseidonCodeHash: codehash.EmptyPoseidonCodeHash.Bytes(),
			CodeSize:         0,
		})

		accounts[i] = data
	}
	return addresses, accounts
}

// spongeDb is a dummy db backend which accumulates writes in a sponge
type spongeDb struct {
	sponge  hash.Hash
	id      string
	journal []string
}

func (s *spongeDb) Has(key []byte) (bool, error)             { panic("implement me") }
func (s *spongeDb) Get(key []byte) ([]byte, error)           { return nil, errors.New("no such elem") }
func (s *spongeDb) Delete(key []byte) error                  { panic("implement me") }
func (s *spongeDb) NewBatch() ethdb.Batch                    { return &spongeBatch{s} }
func (s *spongeDb) Stat(property string) (string, error)     { panic("implement me") }
func (s *spongeDb) Compact(start []byte, limit []byte) error { panic("implement me") }
func (s *spongeDb) Close() error                             { return nil }
func (s *spongeDb) Put(key []byte, value []byte) error {
	valbrief := value
	if len(valbrief) > 8 {
		valbrief = valbrief[:8]
	}
	s.journal = append(s.journal, fmt.Sprintf("%v: PUT([%x...], [%d bytes] %x...)\n", s.id, key[:8], len(value), valbrief))
	s.sponge.Write(key)
	s.sponge.Write(value)
	return nil
}
func (s *spongeDb) NewIterator(prefix []byte, start []byte) ethdb.Iterator { panic("implement me") }

// spongeBatch is a dummy batch which immediately writes to the underlying spongedb
type spongeBatch struct {
	db *spongeDb
}

func (b *spongeBatch) Put(key, value []byte) error {
	b.db.Put(key, value)
	return nil
}
func (b *spongeBatch) Delete(key []byte) error             { panic("implement me") }
func (b *spongeBatch) ValueSize() int                      { return 100 }
func (b *spongeBatch) Write() error                        { return nil }
func (b *spongeBatch) Reset()                              {}
func (b *spongeBatch) Replay(w ethdb.KeyValueWriter) error { return nil }

// TestCommitSequence tests that the trie.Commit operation writes the elements of the trie
// in the expected order, and calls the callbacks in the expected order.
// The test data was based on the 'master' code, and is basically random. It can be used
// to check whether changes to the trie modifies the write order or data in any way.
func TestCommitSequence(t *testing.T) {
	t.Skip("zk-trie writes database on each trie update and commit does nothing.")
	for i, tc := range []struct {
		count              int
		expWriteSeqHash    []byte
		expCallbackSeqHash []byte
	}{
		{20, common.FromHex("7b908cce3bc16abb3eac5dff6c136856526f15225f74ce860a2bec47912a5492"),
			common.FromHex("fac65cd2ad5e301083d0310dd701b5faaff1364cbe01cdbfaf4ec3609bb4149e")},
		{200, common.FromHex("55791f6ec2f83fee512a2d3d4b505784fdefaea89974e10440d01d62a18a298a"),
			common.FromHex("5ab775b64d86a8058bb71c3c765d0f2158c14bbeb9cb32a65eda793a7e95e30f")},
		{2000, common.FromHex("ccb464abf67804538908c62431b3a6788e8dc6dee62aff9bfe6b10136acfceac"),
			common.FromHex("b908adff17a5aa9d6787324c39014a74b04cef7fba6a92aeb730f48da1ca665d")},
	} {
		addresses, accounts := makeAccounts(tc.count)
		// This spongeDb is used to check the sequence of disk-db-writes
		s := &spongeDb{sponge: sha3.NewLegacyKeccak256()}
		db := NewDatabase(s)
		trie, _ := New(common.Hash{}, db)
		// Another sponge is used to check the callback-sequence
		callbackSponge := sha3.NewLegacyKeccak256()
		// Fill the trie with elements
		for i := 0; i < tc.count; i++ {
			trie.Update(crypto.Keccak256(addresses[i][:]), accounts[i])
		}
		// Flush trie -> database
		root, _, _ := trie.Commit(nil)
		// Flush memdb -> disk (sponge)
		db.Commit(root, false, func(c common.Hash) {
			// And spongify the callback-order
			callbackSponge.Write(c[:])
		})
		if got, exp := s.sponge.Sum(nil), tc.expWriteSeqHash; !bytes.Equal(got, exp) {
			t.Errorf("test %d, disk write sequence wrong:\ngot %x exp %x\n", i, got, exp)
		}
		if got, exp := callbackSponge.Sum(nil), tc.expCallbackSeqHash; !bytes.Equal(got, exp) {
			t.Errorf("test %d, call back sequence wrong:\ngot: %x exp %x\n", i, got, exp)
		}
	}
}

// TestCommitSequenceRandomBlobs is identical to TestCommitSequence
// but uses random blobs instead of 'accounts'
func TestCommitSequenceRandomBlobs(t *testing.T) {
	t.Skip("zk-trie writes database on each trie update and commit does nothing.")
	for i, tc := range []struct {
		count              int
		expWriteSeqHash    []byte
		expCallbackSeqHash []byte
	}{
		{20, common.FromHex("8e4a01548551d139fa9e833ebc4e66fc1ba40a4b9b7259d80db32cff7b64ebbc"),
			common.FromHex("450238d73bc36dc6cc6f926987e5428535e64be403877c4560e238a52749ba24")},
		{200, common.FromHex("6869b4e7b95f3097a19ddb30ff735f922b915314047e041614df06958fc50554"),
			common.FromHex("0ace0b03d6cb8c0b82f6289ef5b1a1838306b455a62dafc63cada8e2924f2550")},
		{2000, common.FromHex("444200e6f4e2df49f77752f629a96ccf7445d4698c164f962bbd85a0526ef424"),
			common.FromHex("117d30dafaa62a1eed498c3dfd70982b377ba2b46dd3e725ed6120c80829e518")},
	} {
		prng := rand.New(rand.NewSource(int64(i)))
		// This spongeDb is used to check the sequence of disk-db-writes
		s := &spongeDb{sponge: sha3.NewLegacyKeccak256()}
		db := NewDatabase(s)
		trie, _ := New(common.Hash{}, db)
		// Another sponge is used to check the callback-sequence
		callbackSponge := sha3.NewLegacyKeccak256()
		// Fill the trie with elements
		for i := 0; i < tc.count; i++ {
			key := make([]byte, 32)
			var val []byte
			// 50% short elements, 50% large elements
			if prng.Intn(2) == 0 {
				val = make([]byte, 1+prng.Intn(32))
			} else {
				val = make([]byte, 1+prng.Intn(4096))
			}
			prng.Read(key)
			prng.Read(val)
			trie.Update(key, val)
		}
		// Flush trie -> database
		root, _, _ := trie.Commit(nil)
		// Flush memdb -> disk (sponge)
		db.Commit(root, false, func(c common.Hash) {
			// And spongify the callback-order
			callbackSponge.Write(c[:])
		})
		if got, exp := s.sponge.Sum(nil), tc.expWriteSeqHash; !bytes.Equal(got, exp) {
			t.Fatalf("test %d, disk write sequence wrong:\ngot %x exp %x\n", i, got, exp)
		}
		if got, exp := callbackSponge.Sum(nil), tc.expCallbackSeqHash; !bytes.Equal(got, exp) {
			t.Fatalf("test %d, call back sequence wrong:\ngot: %x exp %x\n", i, got, exp)
		}
	}
}

func TestCommitSequenceStackTrie(t *testing.T) {
	//TODO(kevinyum): re-enable after stack trie is implemented
	t.Skip("re-enable after stack trie is implemented.")
	for count := 1; count < 200; count++ {
		prng := rand.New(rand.NewSource(int64(count)))
		// This spongeDb is used to check the sequence of disk-db-writes
		s := &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "a"}
		db := NewDatabase(s)
		trie, _ := New(common.Hash{}, db)
		// Another sponge is used for the stacktrie commits
		stackTrieSponge := &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "b"}
		stTrie := NewStackTrie(stackTrieSponge)
		// Fill the trie with elements
		for i := 1; i < count; i++ {
			// For the stack trie, we need to do inserts in proper order
			key := make([]byte, 32)
			binary.BigEndian.PutUint64(key, uint64(i))
			var val []byte
			// 50% short elements, 50% large elements
			if prng.Intn(2) == 0 {
				val = make([]byte, 1+prng.Intn(32))
			} else {
				val = make([]byte, 1+prng.Intn(1024))
			}
			prng.Read(val)
			trie.TryUpdate(key, val)
			stTrie.TryUpdate(key, val)
		}
		// Flush trie -> database
		root, _, _ := trie.Commit(nil)
		// Flush memdb -> disk (sponge)
		db.Commit(root, false, nil)
		// And flush stacktrie -> disk
		stRoot, err := stTrie.Commit()
		if err != nil {
			t.Fatalf("Failed to commit stack trie %v", err)
		}
		if stRoot != root {
			t.Fatalf("root wrong, got %x exp %x", stRoot, root)
		}
		if got, exp := stackTrieSponge.sponge.Sum(nil), s.sponge.Sum(nil); !bytes.Equal(got, exp) {
			// Show the journal
			t.Logf("Expected:")
			for i, v := range s.journal {
				t.Logf("op %d: %v", i, v)
			}
			t.Logf("Stacktrie:")
			for i, v := range stackTrieSponge.journal {
				t.Logf("op %d: %v", i, v)
			}
			t.Fatalf("test %d, disk write sequence wrong:\ngot %x exp %x\n", count, got, exp)
		}
	}
}

// TestCommitSequenceSmallRoot tests that a trie which is essentially only a
// small (<32 byte) shortnode with an included value is properly committed to a
// database.
// This case might not matter, since in practice, all keys are 32 bytes, which means
// that even a small trie which contains a leaf will have an extension making it
// not fit into 32 bytes, rlp-encoded. However, it's still the correct thing to do.
func TestCommitSequenceSmallRoot(t *testing.T) {
	//TODO(kevinyum): re-enable after stack trie is implemented
	t.Skip("re-enable after stack trie is implemented.")
	s := &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "a"}
	db := NewDatabase(s)
	trie, _ := New(common.Hash{}, db)
	// Another sponge is used for the stacktrie commits
	stackTrieSponge := &spongeDb{sponge: sha3.NewLegacyKeccak256(), id: "b"}
	stTrie := NewStackTrie(stackTrieSponge)
	// Add a single small-element to the trie(s)
	key := make([]byte, 5)
	key[0] = 1
	trie.TryUpdate(key, []byte{0x1})
	stTrie.TryUpdate(key, []byte{0x1})
	// Flush trie -> database
	root, _, _ := trie.Commit(nil)
	// Flush memdb -> disk (sponge)
	db.Commit(root, false, nil)
	// And flush stacktrie -> disk
	stRoot, err := stTrie.Commit()
	if err != nil {
		t.Fatalf("Failed to commit stack trie %v", err)
	}
	if stRoot != root {
		t.Fatalf("root wrong, got %x exp %x", stRoot, root)
	}
	fmt.Printf("root: %x\n", stRoot)
	if got, exp := stackTrieSponge.sponge.Sum(nil), s.sponge.Sum(nil); !bytes.Equal(got, exp) {
		t.Fatalf("test, disk write sequence wrong:\ngot %x exp %x\n", got, exp)
	}
}

func getString(trie *Trie, k string) []byte {
	return trie.Get([]byte(k))
}

func updateString(trie *Trie, k, v string) {
	trie.Update([]byte(k), []byte(v))
}

func deleteString(trie *Trie, k string) {
	trie.Delete([]byte(k))
}
