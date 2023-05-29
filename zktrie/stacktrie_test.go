// Copyright 2021 The go-ethereum Authors
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
	"fmt"
	"math/big"
	"sort"
	"testing"

	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func TestStackTrieInsertAndHash(t *testing.T) {
	type KeyValueHash struct {
		K string // Hex string for key.
		V string // Value, directly converted to bytes.
		H string // Expected root hash after insert of (K, V) to an existing trie.
	}
	tests := [][]KeyValueHash{
		{ // {0:0, 7:0, f:0}
			{"0000000000000000000000000000000000000000000000000000000000000000", "v_______________________0___0", "0bb2d1db0797580bc8e17ce50122e4f7d128fc89dbbe600cc97724deb72f1fd2"},
			{"7000000000000000000000000000000000000000000000000000000000000000", "v_______________________0___1", "2cdbf9d77e744f5e8415a875388f6e947f7cce832f54afd7c0c80a55d5f1f0ce"},
			{"f000000000000000000000000000000000000000000000000000000000000000", "v_______________________0___2", "01b13aa49c1356ee92ecff03c1c78b726a7e0f3568269ef25a82444d6bbea838"},
		},
		{ // {1:0cc, e:{1:fc, e:fc}}
			{"10cc000000000000000000000000000000000000000000000000000000000000", "v_______________________1___0", "0d5c3d262e87f14577be967fe609b6fc2d5e01239950066f44516f0022965045"},
			{"e1fc000000000000000000000000000000000000000000000000000000000000", "v_______________________1___1", "2e79b9aaf3e929c01f27596ccd3367cb8f13cdfac7441e64c79f0fb425471fc9"},
			{"eefc000000000000000000000000000000000000000000000000000000000000", "v_______________________1___2", "2591a33e008397e115237fcafa5a302a0b3b10a2401aec8656c87f9be4471908"},
		},
		{ // {b:{a:ac, b:ac}, d:acc}
			{"baac000000000000000000000000000000000000000000000000000000000000", "v_______________________2___0", "224d3eefed28574c389ac4ab2092aeef11c527245d65c24500acc3ba642451df"},
			{"bbac000000000000000000000000000000000000000000000000000000000000", "v_______________________2___1", "278ac17a9c0fddad28b003fd41099dbc285522a60e2343b382393ba339dd56ae"},
			{"dacc000000000000000000000000000000000000000000000000000000000000", "v_______________________2___2", "007f4646d50fd8be863c6a569a8b00c25a3fb4f39d0a7b206f7c4b85c04ad67f"},
		},
		{ // {0:0cccc, 2:456{0:0, 2:2}
			{"00cccc0000000000000000000000000000000000000000000000000000000000", "v_______________________3___0", "1acfb852cc9f1cd558a9e9501f5aed197bed164b3d4703f0fd7a1fff55d6cf7d"},
			{"2456000000000000000000000000000000000000000000000000000000000000", "v_______________________3___1", "290335cca308495cb92da0109b3c22905699cc08e59216f4a6bee997543991ea"},
			{"2456220000000000000000000000000000000000000000000000000000000000", "v_______________________3___2", "074e0f3cb64f84a806fb7d9a4204b3104300b4e41ad9668b3c7c6932e416e2a1"},
		},
		{ // {1:4567{1:1c, 3:3c}, 3:0cccccc}
			{"1456711c00000000000000000000000000000000000000000000000000000000", "v_______________________4___0", "230c358f15fc1ba599d5350a55c06218f913392bf3354d3d3ef780f821329e0e"},
			{"1456733c00000000000000000000000000000000000000000000000000000000", "v_______________________4___1", "1e05586e5b9a69aa2d8083fc4ef90a9c42cfedc10e62321c9ad2968e9e6dedbe"},
			{"30cccccc00000000000000000000000000000000000000000000000000000000", "v_______________________4___2", "10d092fd0663ef69c31c1496c6c930fd65c0985809eda207b2776a5847ceb07f"},
		},
		{ // 8800{1:f, 2:e, 3:d}
			{"88001f0000000000000000000000000000000000000000000000000000000000", "v_______________________5___0", "088ecaf9fd1a95c9262b9aa4efd37ce00ee94f9ffb4654069c9fd00633e32af0"},
			{"88002e0000000000000000000000000000000000000000000000000000000000", "v_______________________5___1", "0691165aeeff81ac0267e1699e987d70faaf1f5c9b96db536d63a4bb0dba76bb"},
			{"88003d0000000000000000000000000000000000000000000000000000000000", "v_______________________5___2", "2b6c42b766dda7790d1da6fe6299fa46467bc429f98e68ac2c7832ef9020a37f"},
		},
		{ // 0{1:fc, 2:ec, 4:dc}
			{"01fc000000000000000000000000000000000000000000000000000000000000", "v_______________________6___0", "02e0528ec51aca4010a7c0cf3982ece78460c27da10826f4fdd975d4cd0c9e7b"},
			{"02ec000000000000000000000000000000000000000000000000000000000000", "v_______________________6___1", "1f6cbf0501a75753eb7556a42d4f792489c2097f728265f11a4cc3a884c4a019"},
			{"04dc000000000000000000000000000000000000000000000000000000000000", "v_______________________6___2", "19029bf41c033218a3480215dabee633cc6cb2b39bf99182f4def82656e6d5b0"},
		},
		{ // f{0:fccc, f:ff{0:f, f:f}}
			{"f0fccc0000000000000000000000000000000000000000000000000000000000", "v_______________________7___0", "1a2bcea2350318178d05f06a7c45270c0e711195de80b52ec03baaf464a8474c"},
			{"ffff0f0000000000000000000000000000000000000000000000000000000000", "v_______________________7___1", "2263056aa1fd4f3e18fb26b422a6fece59c65e3367ff24c47c1de5e643cd7866"},
			{"ffffff0000000000000000000000000000000000000000000000000000000000", "v_______________________7___2", "201d00bad6897f7a09b27111830fffb060272c29801d2f94c8efa7a89aa29526"},
		},
		{ // ff{0:f{0:f, f:f}, f:fcc}
			{"ff0f0f0000000000000000000000000000000000000000000000000000000000", "v_______________________8___0", "1d4a8c374754a86ae667aa0c3a02b2e9126d635972582ec906b39ca4e9e621b8"},
			{"ff0fff0000000000000000000000000000000000000000000000000000000000", "v_______________________8___1", "1ac82e16e78772d0db89e575f4fd1c4e3654338ca9feecfdb9ecf5898b2a04db"},
			{"ffffcc0000000000000000000000000000000000000000000000000000000000", "v_______________________8___2", "1c4879e495d1d0f074ba9675fdbae54878ed7c6073d87e342b129d07515068f2"},
		},
	}
	st := NewStackTrie(nil)
	for i, test := range tests {
		// The StackTrie does not allow Insert(), Hash(), Insert(), ...
		// so we will create new trie for every sequence length of inserts.
		for l := 1; l <= len(test); l++ {
			st.Reset()
			for j := 0; j < l; j++ {
				kv := &test[j]
				if err := st.TryUpdate(common.FromHex(kv.K), []byte(kv.V)); err != nil {
					t.Fatal(err)
				}
			}
			expected := common.HexToHash(test[l-1].H)
			if h := st.Hash(); h != expected {
				t.Errorf("%d(%d): root hash mismatch: %x, expected %x", i, l, h, expected)
			}
		}
	}
}

func TestKeyRange(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	key := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e500")
	value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a394cf40d0d2b44f2b66e07c")

	err := nt.TryUpdate(key, value)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	err = st.TryUpdate(key, value)
	if err != nil {
		t.Errorf("%v\n", err)
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestInsertOrder(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5a00", V: "9496f4ec2bf9dab484cac6be589e8417d84781be08"},
		{K: "40edb63a35fcf86c08022722aa3287cdd36440d671b4918131b2514795fefa00", V: "01"},
		{K: "b10e2d527612073b26eecdfd717e6a320cf44b4afac2b0732d9fcbe2b7fa0c00", V: "947a30f7736e48d6599356464ba4c150d8da0302ff"},
		{K: "c2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f800", V: "02"},
	}

	for _, kv := range kvs {
		fmt.Printf("%v\n", common.FromHex(kv.K))
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestUpdateAccount(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	account := new(types.StateAccount)
	account.Nonce = 3
	account.Balance = big.NewInt(12345667891011)
	account.Root = common.Hash{}
	account.Root.SetBytes(common.FromHex("12345"))
	account.KeccakCodeHash = common.FromHex("678910")
	account.PoseidonCodeHash = common.FromHex("1112131415")

	key := common.FromHex("405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5a00")
	st.UpdateAccount(key, account)
	nt.UpdateAccount(key, account)

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

func TestValLength56(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))

	//leaf := common.FromHex("290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563")
	//value := common.FromHex("94cf40d0d2b44f2b66e07cace1372ca42b73cf21a3")
	kvs := []struct {
		K string
		V string
	}{
		{K: "405787fa12a823e0f2b7631cc41b3ba8828b3321ca811111fa75cd3aa3bb5a00", V: "1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111"},
	}

	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}

	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateSmallNodes tests a case where the leaves are small (both key and value),
// which causes a lot of node-within-node. This case was found via fuzzing.
func TestUpdateSmallNodes(t *testing.T) {
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	kvs := []struct {
		K string
		V string
	}{
		{"63303030", "3041"}, // stacktrie.Update
		{"65", "3000"},       // stacktrie.Update
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestUpdateVariableKeys contains a case which stacktrie fails: when keys of different
// sizes are used, and the second one has the same prefix as the first, then the
// stacktrie fails, since it's unable to 'expand' on an already added leaf.
// For all practical purposes, this is fine, since keys are fixed-size length
// in account and storage tries.
//
// The test is marked as 'skipped', and exists just to have the behaviour documented.
// This case was found via fuzzing.
func TestUpdateVariableKeys(t *testing.T) {
	t.SkipNow()
	st := NewStackTrie(nil)
	nt, _ := New(common.Hash{}, NewDatabase(memorydb.New()))
	kvs := []struct {
		K string
		V string
	}{
		{"0x33303534636532393561313031676174", "303030"},
		{"0x3330353463653239356131303167617430", "313131"},
	}
	for _, kv := range kvs {
		nt.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
		st.TryUpdate(common.FromHex(kv.K), common.FromHex(kv.V))
	}
	if nt.Hash() != st.Hash() {
		t.Fatalf("error %x != %x", st.Hash(), nt.Hash())
	}
}

// TestStacktrieNotModifyValues checks that inserting blobs of data into the
// stacktrie does not mutate the blobs
func TestStacktrieNotModifyValues(t *testing.T) {
	st := NewStackTrie(nil)
	{ // Test a very small trie
		// Give it the value as a slice with large backing alloc,
		// so if the stacktrie tries to append, it won't have to realloc
		value := make([]byte, 1, 100)
		value[0] = 0x2
		want := common.CopyBytes(value)
		st.TryUpdate([]byte{0x01}, value)
		st.Hash()
		if have := value; !bytes.Equal(have, want) {
			t.Fatalf("tiny trie: have %#x want %#x", have, want)
		}
		st = NewStackTrie(nil)
	}
	// Test with a larger trie
	keyB := big.NewInt(1)
	keyDelta := big.NewInt(1)
	var vals [][]byte
	getValue := func(i int) []byte {
		if i%2 == 0 { // large
			return crypto.Keccak256(big.NewInt(int64(i)).Bytes())
		} else { //small
			return big.NewInt(int64(i)).Bytes()
		}
	}
	for i := 0; i < 1000; i++ {
		key := common.BigToHash(keyB)
		keyBytesInRange := append(key.Bytes()[1:], 0)
		value := getValue(i)
		err := st.TryUpdate(keyBytesInRange, value)
		if err != nil {
			t.Fatal(err)
		}
		vals = append(vals, value)
		keyB = keyB.Add(keyB, keyDelta)
		keyDelta.Add(keyDelta, common.Big1)
	}
	st.Hash()
	for i := 0; i < 1000; i++ {
		want := getValue(i)

		have := vals[i]
		if !bytes.Equal(have, want) {
			t.Fatalf("item %d, have %#x want %#x", i, have, want)
		}

	}
}

func randomSortedKV(n int) []kv {
	kvs := make([]kv, n)
	for i := 0; i < n; i++ {
		key := make([]byte, 32)
		binary.BigEndian.PutUint64(key[16:], uint64(i))
		kvs[i].k = crypto.PoseidonSecure(key)
		kvs[i].v = []byte("v")
	}
	sort.SliceStable(kvs, func(i, j int) bool {
		return bytes.Compare(kvs[i].k, kvs[j].k) < 0
	})
	return kvs
}

func BenchmarkStacktrieUpdateFixedSize(b *testing.B) {
	for _, tc := range []struct {
		name string
		size int
	}{
		{"1", 1},
		{"10", 10},
		{"100", 100},
		{"1K", 1000},
		{"10K", 10000},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			kvs := randomSortedKV(tc.size)
			b.ResetTimer()
			for n := 0; n < b.N; n++ {
				db := rawdb.NewMemoryDatabase()

				st := NewStackTrie(db)

				for _, it := range kvs {
					st.Update(it.k, it.v)
				}

				st.Hash()
			}
		})
	}
}
