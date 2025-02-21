package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethdb/leveldb"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/trie"
)

var accountsDone atomic.Uint64
var trieCheckers = make(chan struct{}, runtime.GOMAXPROCS(0)*4)

type dbs struct {
	zkDb  *leveldb.Database
	mptDb *leveldb.Database
}

func main() {
	var (
		mptDbPath = flag.String("mpt-db", "", "path to the MPT node DB")
		zkDbPath  = flag.String("zk-db", "", "path to the ZK node DB")
		mptRoot   = flag.String("mpt-root", "", "root hash of the MPT node")
		zkRoot    = flag.String("zk-root", "", "root hash of the ZK node")
		paranoid  = flag.Bool("paranoid", false, "verifies all node contents against their expected hash")
	)
	flag.Parse()

	zkDb, err := leveldb.New(*zkDbPath, 1024, 128, "", true)
	panicOnError(err, "", "failed to open zk db")
	mptDb, err := leveldb.New(*mptDbPath, 1024, 128, "", true)
	panicOnError(err, "", "failed to open mpt db")

	zkRootHash := common.HexToHash(*zkRoot)
	mptRootHash := common.HexToHash(*mptRoot)

	for i := 0; i < runtime.GOMAXPROCS(0)*4; i++ {
		trieCheckers <- struct{}{}
	}

	checkTrieEquality(&dbs{
		zkDb:  zkDb,
		mptDb: mptDb,
	}, zkRootHash, mptRootHash, "", checkAccountEquality, true, *paranoid)

	for i := 0; i < runtime.GOMAXPROCS(0)*4; i++ {
		<-trieCheckers
	}
}

func panicOnError(err error, label, msg string) {
	if err != nil {
		panic(fmt.Sprint(label, " error: ", msg, " ", err))
	}
}

func dup(s []byte) []byte {
	return append([]byte{}, s...)
}
func checkTrieEquality(dbs *dbs, zkRoot, mptRoot common.Hash, label string, leafChecker func(string, *dbs, []byte, []byte, bool), top, paranoid bool) {
	zkTrie, err := trie.NewZkTrie(zkRoot, trie.NewZktrieDatabaseFromTriedb(trie.NewDatabaseWithConfig(dbs.zkDb, &trie.Config{Preimages: true})))
	panicOnError(err, label, "failed to create zk trie")
	mptTrie, err := trie.NewSecureNoTracer(mptRoot, trie.NewDatabaseWithConfig(dbs.mptDb, &trie.Config{Preimages: true}))
	panicOnError(err, label, "failed to create mpt trie")

	mptLeafCh := loadMPT(mptTrie, top)
	zkLeafCh := loadZkTrie(zkTrie, top, paranoid)

	mptLeafMap := <-mptLeafCh
	zkLeafMap := <-zkLeafCh

	if len(mptLeafMap) != len(zkLeafMap) {
		panic(fmt.Sprintf("%s MPT and ZK trie leaf count mismatch: MPT: %d, ZK: %d", label, len(mptLeafMap), len(zkLeafMap)))
	}

	for preimageKey, zkValue := range zkLeafMap {
		if top {
			// ZkTrie pads preimages with 0s to make them 32 bytes.
			// So we might need to clear those zeroes here since we need 20 byte addresses at top level (ie state trie)
			if len(preimageKey) > 20 {
				for _, b := range []byte(preimageKey)[20:] {
					if b != 0 {
						panic(fmt.Sprintf("%s padded byte is not 0 (preimage %s)", label, hex.EncodeToString([]byte(preimageKey))))
					}
				}
				preimageKey = preimageKey[:20]
			}
		} else if len(preimageKey) != 32 {
			// storage leafs should have 32 byte keys, pad them if needed
			zeroes := make([]byte, 32)
			copy(zeroes, []byte(preimageKey))
			preimageKey = string(zeroes)
		}

		mptKey := crypto.Keccak256([]byte(preimageKey))
		mptVal, ok := mptLeafMap[string(mptKey)]
		if !ok {
			panic(fmt.Sprintf("%s key %s (preimage %s) not found in mpt", label, hex.EncodeToString(mptKey), hex.EncodeToString([]byte(preimageKey))))
		}

		leafChecker(fmt.Sprintf("%s key: %s", label, hex.EncodeToString([]byte(preimageKey))), dbs, zkValue, mptVal, paranoid)
	}
}

func checkAccountEquality(label string, dbs *dbs, zkAccountBytes, mptAccountBytes []byte, paranoid bool) {
	mptAccount := &types.StateAccount{}
	panicOnError(rlp.DecodeBytes(mptAccountBytes, mptAccount), label, "failed to decode mpt account")
	zkAccount, err := types.UnmarshalStateAccount(zkAccountBytes)
	panicOnError(err, label, "failed to decode zk account")

	if mptAccount.Nonce != zkAccount.Nonce {
		panic(fmt.Sprintf("%s nonce mismatch: zk: %d, mpt: %d", label, zkAccount.Nonce, mptAccount.Nonce))
	}

	if mptAccount.Balance.Cmp(zkAccount.Balance) != 0 {
		panic(fmt.Sprintf("%s balance mismatch: zk: %s, mpt: %s", label, zkAccount.Balance.String(), mptAccount.Balance.String()))
	}

	if !bytes.Equal(mptAccount.KeccakCodeHash, zkAccount.KeccakCodeHash) {
		panic(fmt.Sprintf("%s code hash mismatch: zk: %s, mpt: %s", label, hex.EncodeToString(zkAccount.KeccakCodeHash), hex.EncodeToString(mptAccount.KeccakCodeHash)))
	}

	if (zkAccount.Root == common.Hash{}) != (mptAccount.Root == types.EmptyRootHash) {
		panic(fmt.Sprintf("%s empty account root mismatch", label))
	} else if zkAccount.Root != (common.Hash{}) {
		zkRoot := common.BytesToHash(zkAccount.Root[:])
		mptRoot := common.BytesToHash(mptAccount.Root[:])
		<-trieCheckers
		go func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Println(p)
					os.Exit(1)
				}
			}()

			checkTrieEquality(dbs, zkRoot, mptRoot, label, checkStorageEquality, false, paranoid)
			accountsDone.Add(1)
			fmt.Println("Accounts done:", accountsDone.Load())
			trieCheckers <- struct{}{}
		}()
	} else {
		accountsDone.Add(1)
		fmt.Println("Accounts done:", accountsDone.Load())
	}
}

func checkStorageEquality(label string, _ *dbs, zkStorageBytes, mptStorageBytes []byte, _ bool) {
	zkValue := common.BytesToHash(zkStorageBytes)
	_, content, _, err := rlp.Split(mptStorageBytes)
	panicOnError(err, label, "failed to decode mpt storage")
	mptValue := common.BytesToHash(content)
	if !bytes.Equal(zkValue[:], mptValue[:]) {
		panic(fmt.Sprintf("%s storage mismatch: zk: %s, mpt: %s", label, zkValue.Hex(), mptValue.Hex()))
	}
}

func loadMPT(mptTrie *trie.SecureTrie, parallel bool) chan map[string][]byte {
	startKey := make([]byte, 32)
	workers := 1 << 5
	if !parallel {
		workers = 1
	}
	step := byte(0xFF) / byte(workers)

	mptLeafMap := make(map[string][]byte, 1000)
	var mptLeafMutex sync.Mutex

	var mptWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		startKey[0] = byte(i) * step
		trieIt := trie.NewIterator(mptTrie.NodeIterator(startKey))

		mptWg.Add(1)
		go func() {
			defer mptWg.Done()
			for trieIt.Next() {
				if parallel {
					mptLeafMutex.Lock()
				}

				if _, ok := mptLeafMap[string(trieIt.Key)]; ok {
					mptLeafMutex.Unlock()
					break
				}

				mptLeafMap[string(dup(trieIt.Key))] = dup(trieIt.Value)

				if parallel {
					mptLeafMutex.Unlock()
				}

				if parallel && len(mptLeafMap)%10000 == 0 {
					fmt.Println("MPT Accounts Loaded:", len(mptLeafMap))
				}
			}
		}()
	}

	respChan := make(chan map[string][]byte)
	go func() {
		mptWg.Wait()
		respChan <- mptLeafMap
	}()
	return respChan
}

func loadZkTrie(zkTrie *trie.ZkTrie, parallel, paranoid bool) chan map[string][]byte {
	zkLeafMap := make(map[string][]byte, 1000)
	var zkLeafMutex sync.Mutex
	zkDone := make(chan map[string][]byte)
	go func() {
		zkTrie.CountLeaves(func(key, value []byte) {
			preimageKey := zkTrie.GetKey(key)
			if len(preimageKey) == 0 {
				panic(fmt.Sprintf("preimage not found zk trie %s", hex.EncodeToString(key)))
			}

			if parallel {
				zkLeafMutex.Lock()
			}

			zkLeafMap[string(dup(preimageKey))] = value

			if parallel {
				zkLeafMutex.Unlock()
			}

			if parallel && len(zkLeafMap)%10000 == 0 {
				fmt.Println("ZK Accounts Loaded:", len(zkLeafMap))
			}
		}, parallel, paranoid)
		zkDone <- zkLeafMap
	}()
	return zkDone
}
