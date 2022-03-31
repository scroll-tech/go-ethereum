package core

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/trie"
)

type proofList [][]byte

func (n *proofList) Put(key []byte, value []byte) error {
	*n = append(*n, value)
	return nil
}

func (n *proofList) Delete(key []byte) error {
	panic("not supported")
}

func proofListFromString(proofs []string) (proofList, error) {
	var out proofList
	for _, str := range proofs {
		buf, err := hexutil.Decode(str)
		if err != nil {
			return nil, err
		}
		out = append(out, buf)
	}
	return out, nil
}

func proofListFromHex(proofs []hexutil.Bytes) proofList {
	var out proofList
	for _, buf := range proofs {
		out = append(out, buf)
	}
	return out
}

func decodeProofForAccounts(proof proofList, db *memorydb.Database, accounts map[string]*types.StateAccount) {
	for _, buf := range proof {

		n, err := trie.DecodeSMTProof(buf)
		if err != nil {
			log.Warn("decode proof string fail", "error", err)
		} else if n != nil {
			k, err := n.Key()
			if err != nil {
				log.Warn("node has no valid key", "error", err)
			} else {
				//notice: must consistent with trie/merkletree.go
				bt := k[:]
				db.Put(bt, buf)
				if n.Type == trie.NodeTypeLeaf {
					if acc, err := types.UnmarshalStateAccount(n.ValuePreimage); err == nil {
						addrs := common.BytesToAddress(n.KeyPreimage[:common.AddressLength]).String()
						if _, exist := accounts[addrs]; !exist {
							//update an address, even the proof just point to another one (proof of unexist)
							accounts[addrs] = acc
						}

						return
					} else {
						log.Warn("decode account bytes fail", "error", err)
					}
				}
			}
		}

	}
}

func appendSMTPath(lastNode *trie.Node, k []byte, path *types.SMTPath) {
	if bytes.Equal(k, lastNode.ChildL[:]) {
		path.Path = append(path.Path, types.SMTPathNode{
			Value:   k,
			Sibling: lastNode.ChildR[:],
		})
	} else if bytes.Equal(k, lastNode.ChildR[:]) {
		path.Path = append(path.Path, types.SMTPathNode{
			Value:   k,
			Sibling: lastNode.ChildL[:],
		})
	} else {
		panic("Unexpected proof form")
	}
}

// we have a trick here which suppose the proof array include all middle nodes along the
// whole path in sequence, from root to leaf, and return final node
func decodeProofForMPTPath(proof proofList, path *types.SMTPath) *trie.Node {

	var lastNode *trie.Node
	path.KeyPathPart = types.HexInt{big.NewInt(0)}
	keyCounter := big.NewInt(1)

	for _, buf := range proof {
		n, err := trie.DecodeSMTProof(buf)
		if err != nil {
			log.Warn("decode proof string fail", "error", err)
		} else if n != nil {
			k, err := n.Key()
			if err != nil {
				log.Warn("node has no valid key", "error", err)
				return n
			}
			if lastNode == nil {
				//use the copy of REVERSEORDER of k[:]
				path.Root = k.Bytes()
			} else {
				if bytes.Equal(k[:], lastNode.ChildL[:]) {
					path.Path = append(path.Path, types.SMTPathNode{
						Value:   k[:],
						Sibling: lastNode.ChildR[:],
					})
				} else if bytes.Equal(k[:], lastNode.ChildR[:]) {
					path.Path = append(path.Path, types.SMTPathNode{
						Value:   k[:],
						Sibling: lastNode.ChildL[:],
					})
					path.KeyPathPart.Add(path.KeyPathPart.Int, keyCounter)
				} else {
					panic("Unexpected proof form")
				}
				keyCounter.Mul(keyCounter, big.NewInt(2))
			}
			switch n.Type {
			case trie.NodeTypeMiddle:
				lastNode = n
			case trie.NodeTypeLeaf:
				path.Leaf = &types.SMTPathNode{
					//TODO: not sure here should be Bytes (reverse order) or Bytes2
					Value:   n.Entry[1][:],
					Sibling: n.Entry[0][:],
				}
				//sanity check
				keyPart := path.KeyPathPart.Bytes()
				for i, b := range keyPart {
					ri := len(keyPart) - i
					cb := path.Leaf.Sibling[ri-1] //notice the output is little-endian
					if b&cb != b {
						panic(fmt.Errorf("path key not match: part is %x but key is %x", keyPart, []byte(path.Leaf.Sibling[:])))
					}
				}

				return n
			case trie.NodeTypeEmpty:
				//we omit the empty node because it can be derived from the
				//0 hash in parent
				return n
			default:
				panic(fmt.Errorf("unknown node type %d", n.Type))
			}
		}
	}

	panic("Unexpected finished here")
}

type smtProofWriter struct {
	underlayerDb    *memorydb.Database
	tracingSMT      *trie.SecureBinaryTrie
	tracingAccounts map[string]*types.StateAccount

	sstoreBefore    *types.StructLogRes
	currentContract common.Address

	outTrace []*types.StateTrace
}

func newSMTProofWriter(storage *types.StorageRes) (*smtProofWriter, error) {

	underlayerDb := memorydb.New()

	accounts := make(map[string]*types.StateAccount)

	// start with from/to's data
	decodeProofForAccounts(proofListFromHex(storage.ProofFrom), underlayerDb, accounts)
	decodeProofForAccounts(proofListFromHex(storage.ProofTo), underlayerDb, accounts)

	smt, err := trie.NewSecureBinaryTrie(
		*storage.RootBefore,
		trie.NewDatabase(underlayerDb),
	)
	if err != nil {
		return nil, fmt.Errorf("smt create failure: %s", err)
	}

	return &smtProofWriter{
		underlayerDb:    underlayerDb,
		tracingSMT:      smt,
		tracingAccounts: accounts,
		currentContract: storage.ToAddress,
	}, nil
}

func getAccountProof(l *types.StructLogRes) *types.AccountProofWrapper {
	if exData := l.ExtraData; exData == nil {
		return nil
	} else if len(exData.ProofList) == 0 {
		return nil
	} else {
		return exData.ProofList[0]
	}
}

func getStorage(l *types.StructLogRes) *types.StorageProofWrapper {
	if acc := getAccountProof(l); acc == nil {
		return nil
	} else if stg := acc.Storage; stg == nil {
		return nil
	} else {
		return stg
	}
}

func mustGetStorageProof(l *types.StructLogRes) []string {
	ret := getStorage(l)
	if ret == nil {
		panic("No storage proof in log")
	}

	return ret.Proof
}

func verifyAccountNode(addr *common.Address, n *trie.Node) error {

	if n.Type != trie.NodeTypeLeaf {
		return fmt.Errorf("not leaf type")
	} else if !bytes.Equal(n.KeyPreimage[:common.AddressLength], addr.Bytes()) {
		return fmt.Errorf("unexpected address: %s vs %x", addr, n.KeyPreimage[:])
	}

	return nil
}

// update account state, and return the corresponding trace object which
// is still opened for more infos
func (w *smtProofWriter) traceAccountUpdate(addr *common.Address, accDataBefore, accData *types.StateAccount) (*types.StateTrace, error) {

	out := new(types.StateTrace)
	//account trie
	out.Address = addr.Bytes()
	out.AccountPath = [2]*types.SMTPath{{}, {}}
	//fill dummy
	out.AccountUpdate = [2]*types.StateAccountL2{{}, {}}

	if accData != nil {
		out.AccountUpdate[1] = &types.StateAccountL2{
			Nonce:    int(accData.Nonce),
			Balance:  types.HexInt{Int: big.NewInt(0).Set(accData.Balance)},
			CodeHash: accData.CodeHash,
		}
	}
	if accDataBefore != nil {
		out.AccountUpdate[0] = &types.StateAccountL2{
			Nonce:    int(accDataBefore.Nonce),
			Balance:  types.HexInt{Int: big.NewInt(0).Set(accDataBefore.Balance)},
			CodeHash: accDataBefore.CodeHash,
		}
	}

	var proof proofList
	if err := w.tracingSMT.Prove(addr.Bytes32(), 0, &proof); err != nil {
		return nil, fmt.Errorf("prove BEFORE state fail: %s", err)
	}

	nBefore := decodeProofForMPTPath(proof, out.AccountPath[0])
	if accDataBefore != nil {
		if err := verifyAccountNode(addr, nBefore); err != nil {
			return nil, fmt.Errorf("state BEFORE has no valid account: %s", err)
		}
		// we have ensured the nBefore has a key corresponding to the query one
		out.AccountKey = nBefore.Entry[0].Bytes()
	}

	if accData != nil {
		if err := w.tracingSMT.TryUpdateAccount(addr.Bytes32(), accData); err != nil {
			return nil, fmt.Errorf("update smt account state fail: %s", err)
		}
		w.tracingAccounts[addr.String()] = accData
	} else {
		if err := w.tracingSMT.TryDelete(addr.Bytes32()); err != nil {
			return nil, fmt.Errorf("delete smt account state fail: %s", err)
		}
		delete(w.tracingAccounts, addr.String())
	}

	proof = proofList{}
	if err := w.tracingSMT.Prove(addr.Bytes32(), 0, &proof); err != nil {
		return nil, fmt.Errorf("prove AFTER state fail: %s", err)
	}

	nAfter := decodeProofForMPTPath(proof, out.AccountPath[1])
	if accData != nil {
		if err := verifyAccountNode(addr, nAfter); err != nil {
			return nil, fmt.Errorf("state AFTER has no valid account: %s", err)
		}
		if out.AccountKey == nil {
			out.AccountKey = nAfter.Entry[0].Bytes()
		}
		//now accountKey must has been filled
	}

	return out, nil
}

//handleSStore would return nil for a non-op (i.e. SSTORE a value identify to before state)
func (w *smtProofWriter) handleSStore(lBefore *types.StructLogRes, l *types.StructLogRes) (*types.StateTrace, error) {

	if lBefore.Stack == nil || len(*lBefore.Stack) == 0 {
		panic("unexpected log stack for SSTACK")
	}
	storeAddr := hexutil.MustDecodeBig((*lBefore.Stack)[len(*lBefore.Stack)-1])

	log.Debug("handle SSTORE", "pc", l.Pc)

	acc, existed := w.tracingAccounts[w.currentContract.String()]
	if !existed {
		return nil, fmt.Errorf("contract has no %s account for trace", w.currentContract)
	}

	statePath := [2]*types.SMTPath{{}, {}}
	stateUpdate := [2]*types.StateStorageL2{}

	var storageBeforeProof, storageAfterProof proofList
	var err error
	if storageBeforeProof, err = proofListFromString(mustGetStorageProof(lBefore)); err != nil {
		return nil, fmt.Errorf("invalid hex string: %s", err)
	}

	sBefore := decodeProofForMPTPath(storageBeforeProof, statePath[0])
	log.Debug("decode for sstore before", "node", sBefore)
	if sBefore.Type == trie.NodeTypeLeaf && storeAddr.Cmp(big.NewInt(0).SetBytes(sBefore.KeyPreimage[:])) == 0 {
		stateUpdate[0] = &types.StateStorageL2{
			Key:   sBefore.KeyPreimage[:],
			Value: sBefore.ValuePreimage[:],
		}
	} else {
		stateUpdate[0] = &types.StateStorageL2{}
	}

	//sanity check
	if !bytes.Equal(acc.Root[:], statePath[0].Root) {
		panic(fmt.Errorf("unexpected storage root before: [%s] vs [%s]", acc.Root, statePath[0].Root))
	}

	if storageAfterProof, err = proofListFromString(mustGetStorageProof(l)); err != nil {
		return nil, fmt.Errorf("invalid hex string: %s", err)
	}

	sAfter := decodeProofForMPTPath(storageAfterProof, statePath[1])
	log.Debug("decode for sstore after", "node", sAfter)
	if sAfter.Type == trie.NodeTypeLeaf && storeAddr.Cmp(big.NewInt(0).SetBytes(sAfter.KeyPreimage[:])) == 0 {
		stateUpdate[1] = &types.StateStorageL2{
			Key:   sAfter.KeyPreimage[:],
			Value: sAfter.ValuePreimage[:],
		}
	} else if stateUpdate[0].Key != nil {
		// fast detection for possible malformed data
		return nil, fmt.Errorf("not a leaf node after SSTORE")
	}

	//skip non-op SSTORE trace
	if bytes.Equal(statePath[1].Root, statePath[0].Root) {
		return nil, nil
	}

	accAfter := &types.StateAccount{
		Nonce:    acc.Nonce,
		Balance:  acc.Balance,
		CodeHash: acc.CodeHash,
		Root:     common.BytesToHash(statePath[1].Root),
	}

	out, err := w.traceAccountUpdate(&w.currentContract, acc, accAfter)
	if err != nil {
		return nil, fmt.Errorf("update account %s in SSTORE fail: %s", w.currentContract, err)
	}

	// use Bytes, so we obtain big-endian key which should be test from less-significant bit
	out.StateKey = sAfter.Entry[0].Bytes()
	out.StatePath = statePath
	out.StateUpdate = stateUpdate
	return out, nil
}

// Fill smtproof field for execResult
func (w *smtProofWriter) handleLogs(logs []types.StructLogRes) error {
	// now trace every OP which could cause changes on state:
	for i, sLog := range logs {
		switch sLog.Op {
		case "SSTORE":
			if sLog.Storage != nil {
				if w.sstoreBefore != nil {
					log.Warn("wrong layout in SSTORE", "pc", w.sstoreBefore.Pc)
				}
				//the before state
				logCpy := sLog
				w.sstoreBefore = &logCpy
			} else {
				//the after state, can handle (but check before)
				lBefore := w.sstoreBefore
				w.sstoreBefore = nil
				if lBefore == nil || lBefore.Pc != sLog.Pc {
					return fmt.Errorf("unmatch SSTORE log found [%d]", sLog.Pc)
				}

				if t, err := w.handleSStore(lBefore, &sLog); err == nil {
					if t != nil {
						t.Index = i
						// sanity check
						keyRec, _ := hexutil.Decode(lBefore.ExtraData.ProofList[0].Storage.Key)
						if !bytes.Equal(keyRec, t.StateUpdate[1].Key) {
							panic(fmt.Errorf("SSTORE do not have proof corresponding to its record, want %x but has %x", keyRec, []byte(t.StateUpdate[1].Key)))
						}
						w.outTrace = append(w.outTrace, t)
					} else {
						log.Debug("skip non-op SSTORE", "pc", lBefore.Pc)
					}
				} else {
					return fmt.Errorf("handle SSTORE log fail: %s", err)
				}

			}

		default:
		}
	}

	return nil
}

func (w *smtProofWriter) handleAccountCreate(buf []byte) error {
	if buf == nil {
		return nil
	}

	accData, err := types.UnmarshalStateAccount(buf)
	if err != nil {
		return fmt.Errorf("unmarshall created acc fail: %s", err)
	}

	out, err := w.traceAccountUpdate(&w.currentContract, nil, accData)
	if err != nil {
		return fmt.Errorf("update account %s for creation fail: %s", w.currentContract, err)
	}

	out.Index = -1
	out.CommonStateRoot = accData.Root[:]
	w.outTrace = append(w.outTrace, out)

	return nil
}

//finally update account status which is not traced in logs (Nonce added, gasBuy, gasRefund etc)
func (w *smtProofWriter) handlePostTx(accs map[string]hexutil.Bytes) error {

	for acc, buf := range accs {

		accData, err := types.UnmarshalStateAccount(buf)
		if err != nil {
			return fmt.Errorf("unmarshall acc fail: %s", err)
		}

		accDataBefore, existed := w.tracingAccounts[acc]
		// sanity check
		if !existed {
			panic(fmt.Errorf("account %s has not been traced in Log", acc))
		} else if !bytes.Equal(accData.Root[:], accDataBefore.Root[:]) {
			panic(fmt.Errorf("accout %s is not cleaned for state", acc))
		}

		addrBytes, _ := hexutil.Decode(acc)
		addr := common.BytesToAddress(addrBytes)

		out, err := w.traceAccountUpdate(&addr, accDataBefore, accData)
		if err != nil {
			return fmt.Errorf("update account %s fail: %s", addr, err)
		}

		out.Index = -1
		out.CommonStateRoot = accData.Root[:]
		w.outTrace = append(w.outTrace, out)
	}

	return nil
}

func (w *smtProofWriter) txFinal(rootAfter *common.Hash) error {

	root := w.tracingSMT.Hash()

	if !bytes.Equal(rootAfter[:], root[:]) {
		return fmt.Errorf("unmatched root: expected %x but we have %x", rootAfter[:], root)
	}
	return nil
}
