package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/leveldb"
	"github.com/scroll-tech/go-ethereum/rlp"
	"github.com/scroll-tech/go-ethereum/trie"
)

var accountsDone atomic.Uint64
var totalAccounts int
var trieCheckers chan struct{}

type dbs struct {
	zkDb  *leveldb.Database
	mptDb *leveldb.Database
}

func main() {
	var (
		mptDbPath            = flag.String("mpt-db", "", "path to the MPT node DB")
		zkDbPath             = flag.String("zk-db", "", "path to the ZK node DB")
		mptRoot              = flag.String("mpt-root", "", "root hash of the MPT node")
		zkRoot               = flag.String("zk-root", "", "root hash of the ZK node")
		paranoid             = flag.Bool("paranoid", false, "verifies all node contents against their expected hash")
		parallelismMultipler = flag.Int("parallelism-multiplier", 4, "multiplier for the number of parallel workers")
		startFrom            = flag.Int("start-form", 0, "start checking from account at the given index")
	)
	flag.Parse()

	zkDb, err := leveldb.New(*zkDbPath, 1024, 128, "", true)
	panicOnError(err, "", "failed to open zk db")
	mptDb, err := leveldb.New(*mptDbPath, 1024, 128, "", true)
	panicOnError(err, "", "failed to open mpt db")

	zkRootHash := common.HexToHash(*zkRoot)
	mptRootHash := common.HexToHash(*mptRoot)

	numTrieCheckers := runtime.GOMAXPROCS(0) * (*parallelismMultipler)
	trieCheckers = make(chan struct{}, numTrieCheckers)
	for i := 0; i < numTrieCheckers; i++ {
		trieCheckers <- struct{}{}
	}

	done := make(chan struct{})
	totalCheckers := len(trieCheckers)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(time.Minute):
				fmt.Println("Active checkers:", totalCheckers-len(trieCheckers))
			}
		}
	}()
	defer close(done)

	startFromSafe := *startFrom - len(trieCheckers)
	if startFromSafe < 0 {
		startFromSafe = 0
	}
	accountsDone.Add(uint64(startFromSafe))

	checkTrieEquality(&dbs{
		zkDb:  zkDb,
		mptDb: mptDb,
	}, zkRootHash, mptRootHash, "", checkAccountEquality, true, *paranoid, startFromSafe)

	for i := 0; i < numTrieCheckers; i++ {
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
func checkTrieEquality(dbs *dbs, zkRoot, mptRoot common.Hash, label string, leafChecker func(string, *dbs, []byte, []byte, bool), top, paranoid bool, startFrom int) {
	done := make(chan struct{})
	start := time.Now()
	if !top {
		go func() {
			for {
				select {
				case <-done:
					return
				case <-time.After(time.Minute):
					fmt.Println("Checking trie", label, "for", time.Since(start))
				}
			}
		}()
	}
	defer close(done)

	zkTrie, err := trie.NewZkTrie(zkRoot, trie.NewZktrieDatabaseFromTriedb(trie.NewDatabaseWithConfig(dbs.zkDb, &trie.Config{Preimages: true})))
	panicOnError(err, label, "failed to create zk trie")
	mptTrie, err := trie.NewSecureNoTracer(mptRoot, trie.NewDatabaseWithConfig(dbs.mptDb, &trie.Config{Preimages: true}))
	panicOnError(err, label, "failed to create mpt trie")

	mptLeafCh := loadMPT(mptTrie, top)
	zkLeafCh := loadZkTrie(zkTrie, top, paranoid)

	mptLeafs := <-mptLeafCh
	zkLeafs := <-zkLeafCh

	if len(mptLeafs) != len(zkLeafs) {
		panic(fmt.Sprintf("%s MPT and ZK trie leaf count mismatch: MPT: %d, ZK: %d", label, len(mptLeafs), len(zkLeafs)))
	} else if top {
		totalAccounts = len(mptLeafs)
	}

	for i := startFrom; i < len(zkLeafs); i++ {
		zkKv := zkLeafs[i]
		mptKv := mptLeafs[i]
		leafChecker(fmt.Sprintf("%s key: %s", label, hex.EncodeToString([]byte(zkKv.key))), dbs, zkKv.value, mptKv.value, paranoid)
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

			checkTrieEquality(dbs, zkRoot, mptRoot, label, checkStorageEquality, false, paranoid, 0)
			accountsDone.Add(1)
			fmt.Println("Accounts done:", accountsDone.Load(), "/", totalAccounts)
			trieCheckers <- struct{}{}
		}()
	} else {
		accountsDone.Add(1)
		fmt.Println("Accounts done:", accountsDone.Load(), "/", totalAccounts)
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

type kv struct {
	key, value []byte
}

func loadMPT(mptTrie *trie.SecureTrie, top bool) chan []kv {
	startKey := make([]byte, 32)
	workers := 1 << 5
	if !top {
		workers = 1 << 3
	}
	step := byte(256 / workers)

	mptLeafs := make([]kv, 0, 1000)
	var mptLeafMutex sync.Mutex

	var mptWg sync.WaitGroup
	for i := 0; i < workers; i++ {
		startKey[0] = byte(i) * step
		trieIt := trie.NewIterator(mptTrie.NodeIterator(startKey))

		stopKey := (i + 1) * int(step)
		mptWg.Add(1)
		go func() {
			defer mptWg.Done()
			for trieIt.Next() {
				if int(trieIt.Key[0]) >= stopKey {
					break
				}

				preimageKey := mptTrie.GetKey(trieIt.Key)
				if len(preimageKey) == 0 {
					panic(fmt.Sprintf("preimage not found mpt trie %s", hex.EncodeToString(trieIt.Key)))
				}

				mptLeafMutex.Lock()
				mptLeafs = append(mptLeafs, kv{key: preimageKey, value: dup(trieIt.Value)})
				mptLeafMutex.Unlock()
				if top && len(mptLeafs)%10000 == 0 {
					fmt.Println("MPT Accounts Loaded:", len(mptLeafs))
				}
			}
		}()
	}

	respChan := make(chan []kv)
	go func() {
		mptWg.Wait()
		sort.Slice(mptLeafs, func(i, j int) bool {
			return bytes.Compare(mptLeafs[i].key, mptLeafs[j].key) < 0
		})
		respChan <- mptLeafs
	}()
	return respChan
}

func loadZkTrie(zkTrie *trie.ZkTrie, top, paranoid bool) chan []kv {
	zkLeafs := make([]kv, 0, 1000)
	var zkLeafMutex sync.Mutex
	zkDone := make(chan []kv)
	go func() {
		zkTrie.CountLeaves(func(key, value []byte) {
			preimageKey := zkTrie.GetKey(key)
			if len(preimageKey) == 0 {
				panic(fmt.Sprintf("preimage not found zk trie %s", hex.EncodeToString(key)))
			}

			if top {
				// ZkTrie pads preimages with 0s to make them 32 bytes.
				// So we might need to clear those zeroes here since we need 20 byte addresses at top level (ie state trie)
				if len(preimageKey) > 20 {
					for _, b := range []byte(preimageKey)[20:] {
						if b != 0 {
							panic(fmt.Sprintf("padded byte is not 0 (preimage %s)", hex.EncodeToString([]byte(preimageKey))))
						}
					}
					preimageKey = preimageKey[:20]
				}
			} else if len(preimageKey) != 32 {
				// storage leafs should have 32 byte keys, pad them if needed
				zeroes := make([]byte, 32)
				copy(zeroes, []byte(preimageKey))
				preimageKey = zeroes
			}

			zkLeafMutex.Lock()
			zkLeafs = append(zkLeafs, kv{key: preimageKey, value: value})
			zkLeafMutex.Unlock()

			if top && len(zkLeafs)%10000 == 0 {
				fmt.Println("ZK Accounts Loaded:", len(zkLeafs))
			}
		}, top, paranoid)

		sort.Slice(zkLeafs, func(i, j int) bool {
			return bytes.Compare(zkLeafs[i].key, zkLeafs[j].key) < 0
		})
		zkDone <- zkLeafs
	}()
	return zkDone
}
