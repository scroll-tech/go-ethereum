package core

import (
	"bytes"
	"fmt"

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

func decodeProofForAccounts(proof proofList, db *memorydb.Database, accounts map[string]*types.StateAccount) common.Address {
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
						addr := common.BytesToAddress(n.KeyPreimage[:common.AddressLength])
						addrs := addr.String()
						if _, exist := accounts[addrs]; !exist {
							//update an address, even the proof just point to another one (proof of unexist)
							accounts[addrs] = acc
						}

						return addr
					} else {
						log.Warn("decode account bytes fail", "error", err)
					}
				}
			}
		}

	}

	return common.Address{}
}

func appendSMTPath(lastNode *trie.Node, k []byte, path *types.SMTPath) {
	if bytes.Equal(k, lastNode.ChildL[:]) {
		path.Path = append(path.Path, types.SMTPathNode{
			Value:    k,
			Silbling: lastNode.ChildR[:],
		})
	} else if bytes.Equal(k, lastNode.ChildR[:]) {
		path.Path = append(path.Path, types.SMTPathNode{
			Value:    k,
			Silbling: lastNode.ChildL[:],
		})
	} else {
		panic("Unexpected proof form")
	}
}

// we have a trick here which suppose the proof array include all middle nodes along the
// whole path in sequence, from root to leaf, and return final node
func decodeProofForMPTPath(proof proofList, path *types.SMTPath) *trie.Node {

	var lastNode *trie.Node

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
			}
			if n.Type == trie.NodeTypeMiddle {
				if lastNode != nil {
					appendSMTPath(lastNode, k[:], path)
				}
				lastNode = n
			} else {
				path.Path = append(path.Path, types.SMTPathNode{
					Value:    k[:],
					Silbling: make([]byte, common.HashLength),
				})
				return n
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

	smt, err := trie.NewSecureBinaryTrie(
		*storage.RootBefore,
		trie.NewDatabase(underlayerDb),
	)
	if err != nil {
		return nil, fmt.Errorf("smt create failure: %s", err)
	}

	accounts := make(map[string]*types.StateAccount)

	// start with from/to's data
	decodeProofForAccounts(proofListFromHex(storage.ProofFrom), underlayerDb, accounts)
	contractAddr := decodeProofForAccounts(proofListFromHex(storage.ProofTo), underlayerDb, accounts)

	return &smtProofWriter{
		underlayerDb:    underlayerDb,
		tracingSMT:      smt,
		tracingAccounts: accounts,
		currentContract: contractAddr,
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

func getStorageProof(l *types.StructLogRes) []string {
	if acc := getAccountProof(l); acc == nil {
		return nil
	} else if stg := acc.Storage; stg == nil {
		return nil
	} else {
		return stg.Proof
	}
}

func mustGetStorageProof(l *types.StructLogRes) []string {
	ret := getStorageProof(l)
	if ret == nil {
		panic("No storage proof in log")
	}

	return ret
}

func (w *smtProofWriter) handleSStore(lBefore *types.StructLogRes, l *types.StructLogRes) (out *types.StateTrace, err error) {

	log.Debug("handle SSTORE", "pc", l.Pc)

	out = new(types.StateTrace)
	//account trie
	out.AccountPath = [2]*types.SMTPath{{}, {}}
	out.StatePath = [2]*types.SMTPath{{}, {}}
	out.StateUpdate = [2]*types.StateStorageL2{}
	out.AccountUpdate = [2]*types.StateAccountL2{}

	var storageBeforeProof, storageAfterProof proofList
	if storageBeforeProof, err = proofListFromString(mustGetStorageProof(lBefore)); err != nil {
		return nil, fmt.Errorf("invalid hex string: %s", err)
	}

	sBefore := decodeProofForMPTPath(storageBeforeProof, out.StatePath[0])
	if sBefore.Type == trie.NodeTypeLeaf {
		out.StateUpdate[0] = &types.StateStorageL2{
			Key:   sBefore.KeyPreimage[:],
			Value: sBefore.ValuePreimage[:],
		}
	} else {
		out.StateUpdate[0] = &types.StateStorageL2{}
	}
	if k, err := sBefore.Key(); err != nil {
		return nil, fmt.Errorf("invalid stateBefore node key: %s", err)
	} else {
		out.StateKeyBefore = k[:]
	}

	if storageAfterProof, err = proofListFromString(mustGetStorageProof(l)); err != nil {
		return nil, fmt.Errorf("invalid hex string: %s", err)
	}

	sAfter := decodeProofForMPTPath(storageAfterProof, out.StatePath[1])
	if sAfter.Type == trie.NodeTypeLeaf {
		out.StateUpdate[1] = &types.StateStorageL2{
			Key:   sAfter.KeyPreimage[:],
			Value: sAfter.ValuePreimage[:],
		}
	} else {
		return nil, fmt.Errorf("no valid leaf node after SSTORE")
	}
	if k, err := sAfter.Key(); err != nil {
		return nil, fmt.Errorf("invalid stateAfter node key: %s", err)
	} else {
		out.StateKey = k[:]
	}

	var proof proofList
	err = w.tracingSMT.Prove(w.currentContract.Bytes(), 0, &proof)
	if err != nil {
		return nil, fmt.Errorf("prove current address's BEFORE state <%s> fail: %s", w.currentContract, err)
	}

	nBefore := decodeProofForMPTPath(proof, out.AccountPath[0])
	// SSTORE must has account existed
	if nBefore.Type != trie.NodeTypeLeaf {
		return nil, fmt.Errorf("contract has no valid %s account for SSTORE", w.currentContract)
	}

	acc, existed := w.tracingAccounts[w.currentContract.String()]
	if !existed {
		return nil, fmt.Errorf("contract has no %s account for trace", w.currentContract)
	}

	//check
	if !bytes.Equal(acc.Root[:], out.StatePath[0].Root) {
		panic(fmt.Errorf("unexpected storage root before: [%s] vs [%s]", acc.Root, out.StatePath[0].Root))
	}

	//notice in SSTORE Account has no other update
	out.AccountUpdate[0] = &types.StateAccountL2{
		Nonce:    int(acc.Nonce),
		Balance:  acc.Balance.Bytes(),
		CodeHash: acc.CodeHash,
	}

	out.AccountUpdate[1] = out.AccountUpdate[0]

	if k, err := nBefore.Key(); err != nil {
		return nil, fmt.Errorf("invalid accountBefore node key: %s", err)
	} else {
		out.AccountKeyBefore = k[:]
	}

	//update account
	acc.Root = common.BytesToHash(out.StatePath[1].Root)
	if err = w.tracingSMT.TryUpdateAccount(w.currentContract.Bytes32(), acc); err != nil {
		return nil, fmt.Errorf("update smt account state for SSTORE fail: %s", err)
	}

	err = w.tracingSMT.Prove(w.currentContract.Bytes(), 0, &proof)
	if err != nil {
		return nil, fmt.Errorf("prove current address's AFTER state <%s> fail: %s", w.currentContract, err)
	}

	nAfter := decodeProofForMPTPath(proof, out.AccountPath[1])

	if nAfter.Type != trie.NodeTypeLeaf {
		panic(fmt.Errorf("contract has no valid %s account AFTER SSTORE", w.currentContract))
	} else if !bytes.Equal(nAfter.KeyPreimage[:common.AddressLength], w.currentContract.Bytes()) {
		panic(fmt.Errorf("not expected address AFTER SSTORE: %s vs %x", w.currentContract, nAfter.KeyPreimage[:]))
	} else {
		out.Address = w.currentContract.Bytes()
	}

	if k, err := nAfter.Key(); err != nil {
		return nil, fmt.Errorf("invalid accountBefore node key: %s", err)
	} else {
		out.AccountKey = k[:]
	}

	return
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
					t.Index = i
					w.outTrace = append(w.outTrace, t)
				} else {
					return fmt.Errorf("handle SSTORE log fail: %s", err)
				}

			}

		default:
		}
	}

	return nil
}

/*
func (w *smtProofWriter) handleAccountCreate(buf []byte) error {
	accData, err := types.UnmarshalStateAccount(buf)
	if err != nil {
		return fmt.Errorf("unmarshall created acc fail: %s", err)
	}



	return nil
}
*/

//finally update account status which is not traced in logs (Nonce added, gasBuy, gasRefund etc)
func (w *smtProofWriter) handlePostTx(accs map[string]hexutil.Bytes) error {

	for acc, buf := range accs {

		accDataBefore, existed := w.tracingAccounts[acc]
		if !existed {
			return fmt.Errorf("account %s has not been traced in Log", acc)
		}

		addrBytes, _ := hexutil.Decode(acc)

		accData, err := types.UnmarshalStateAccount(buf)
		if err != nil {
			return fmt.Errorf("unmarshall acc fail: %s", err)
		}

		if !bytes.Equal(accData.Root[:], accDataBefore.Root[:]) {
			panic(fmt.Errorf("accout %s is not cleaned for state", acc))
		}

		if accData.Balance.Cmp(accDataBefore.Balance) == 0 &&
			accData.Nonce == accDataBefore.Nonce {

			log.Debug("no update for traced account", "account", acc)
			continue
		}

		out := new(types.StateTrace)
		//account trie
		out.Index = -1
		out.Address = addrBytes
		out.AccountPath = [2]*types.SMTPath{{}, {}}
		out.CommonStateRoot = accData.Root.Bytes()
		out.AccountUpdate = [2]*types.StateAccountL2{
			{
				Nonce:    int(accDataBefore.Nonce),
				Balance:  accDataBefore.Balance.Bytes(),
				CodeHash: accDataBefore.CodeHash,
			},
			{
				Nonce:    int(accData.Nonce),
				Balance:  accData.Balance.Bytes(),
				CodeHash: accData.CodeHash,
			},
		}

		var proof proofList
		if err := w.tracingSMT.Prove(addrBytes, 0, &proof); err != nil {
			return fmt.Errorf("prove <%s>'s BEFORE state fail: %s", acc, err)
		}

		nBefore := decodeProofForMPTPath(proof, out.AccountPath[0])
		// SSTORE must has account existed
		if nBefore.Type != trie.NodeTypeLeaf {
			return fmt.Errorf("state has no valid %s account", acc)
		}

		if err := w.tracingSMT.Prove(addrBytes, 0, &proof); err != nil {
			return fmt.Errorf("prove <%s>'s AFTER state fail: %s", acc, err)
		}

		if err := w.tracingSMT.TryUpdateAccount(common.BytesToAddress(addrBytes).Bytes32(), accData); err != nil {
			return fmt.Errorf("update smt account state fail: %s", err)
		}

		nAfter := decodeProofForMPTPath(proof, out.AccountPath[1])
		// SSTORE must has account existed
		if nAfter.Type != trie.NodeTypeLeaf {
			return fmt.Errorf("state has no valid %s account", acc)
		}

		if !bytes.Equal(nAfter.KeyPreimage[:], nBefore.KeyPreimage[:]) {
			panic(fmt.Errorf("not expected address of before state from trie proof: %x vs %x", nBefore.KeyPreimage[:], nAfter.KeyPreimage[:]))
		} else if !bytes.Equal(nAfter.KeyPreimage[:common.AddressLength], addrBytes) {
			panic(fmt.Errorf("not expected address from trie proof: %x vs %x", addrBytes, nAfter.KeyPreimage[:]))
		}

		if k, err := nAfter.Key(); err != nil {
			return fmt.Errorf("invalid account node key: %s", err)
		} else {
			out.AccountKey = k[:]
		}

		w.tracingAccounts[acc] = accData
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
