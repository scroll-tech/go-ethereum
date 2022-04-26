package core

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/core/types/smt"
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
						log.Warn("decode account bytes fail", "error", err, "data", hexutil.Encode(n.ValuePreimage))
					}
				}
			}
		}

	}
}

// we have a trick here which suppose the proof array include all middle nodes along the
// whole path in sequence, from root to leaf, and return final node
func decodeProofForMPTPath(proof proofList, path *types.SMTPath) *trie.Node {

	var lastNode *trie.Node
	path.KeyPathPart = types.HexInt{Int: big.NewInt(0)}
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
				//notice: use little-endian represent inside Hash ([:] or Bytes2())
				path.Root = k[:]
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

	currentContract common.Address

	outTrace []*types.StateTrace
}

func newSMTProofWriter(storage *types.StorageRes) (*smtProofWriter, error) {

	underlayerDb := memorydb.New()

	accounts := make(map[string]*types.StateAccount)

	// start with from/to's data
	decodeProofForAccounts(proofListFromHex(storage.ProofFrom), underlayerDb, accounts)
	decodeProofForAccounts(proofListFromHex(storage.ProofTo), underlayerDb, accounts)

	smt, err := trie.NewSecure(
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

const (
	posSSTOREBefore = 0
	posSSTOREAfter  = 1 // maybe deprecated later
	posCREATE       = 0
	posCREATEAfter  = 1
	posCALL         = 1
	posSTATICCALL   = 0
)

func getAccountProof(l *types.StructLogRes, pos int) *types.AccountProofWrapper {
	if exData := l.ExtraData; exData == nil {
		return nil
	} else if len(exData.ProofList) < pos {
		return nil
	} else {
		return exData.ProofList[pos]
	}
}

func getAccountDataFromProof(l *types.StructLogRes, pos int) (common.Address, *types.StateAccount) {
	proof := getAccountProof(l, pos)
	if proof == nil {
		return common.Address{}, nil
	}

	return proof.Address, &types.StateAccount{
		Nonce:    proof.Nonce,
		Balance:  (*big.Int)(proof.Balance),
		CodeHash: proof.CodeHash.Bytes(),
	}
}

func getStorage(l *types.StructLogRes, pos int) *types.StorageProofWrapper {
	if acc := getAccountProof(l, pos); acc == nil {
		return nil
	} else if stg := acc.Storage; stg == nil {
		return nil
	} else {
		return stg
	}
}

func mustGetStorageProof(l *types.StructLogRes, pos int) []string {
	ret := getStorage(l, pos)
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

// update traced account state, and return the corresponding trace object which
// is still opened for more infos
// the updated accData state is obtained by a closure which enable it being derived from current status
func (w *smtProofWriter) traceAccountUpdate(addr *common.Address, getAccData func(*types.StateAccount) *types.StateAccount) (*types.StateTrace, error) {

	out := new(types.StateTrace)
	//account trie
	out.Address = addr.Bytes()
	out.AccountPath = [2]*types.SMTPath{{}, {}}
	//fill dummy
	out.AccountUpdate = [2]*types.StateAccountL2{{}, {}}

	accDataBefore, existed := w.tracingAccounts[addr.String()]
	if !existed {
		//sanity check
		panic(fmt.Errorf("code do not add initialized status for account %s", addr))
	}

	var proof proofList
	if err := w.tracingSMT.Prove(addr.Bytes32(), 0, &proof); err != nil {
		return nil, fmt.Errorf("prove BEFORE state fail: %s", err)
	}

	nBefore := decodeProofForMPTPath(proof, out.AccountPath[0])
	if accDataBefore != nil {
		//sanity check
		if err := verifyAccountNode(addr, nBefore); err != nil {
			panic(fmt.Errorf("code fail to trace account status correctly: %s", err))
		}
		if bt := accDataBefore.MarshalBytes(); !bytes.Equal(bt, nBefore.ValuePreimage) {
			panic(fmt.Errorf("code fail to trace account status correctly: %x vs %x", bt, nBefore.ValuePreimage))
		}

		//accH, _ := accDataBefore.Hash()
		//log.Info("sanity check acc before", "addr", addr.String(), "key", nBefore.Entry[1].BigInt().Text(16), "hash", accH.Text(16))

		// we have ensured the nBefore has a key corresponding to the query one
		out.AccountKey = nBefore.Entry[0][:]
		out.AccountUpdate[0] = &types.StateAccountL2{
			Nonce:    int(accDataBefore.Nonce),
			Balance:  types.HexInt{Int: big.NewInt(0).Set(accDataBefore.Balance)},
			CodeHash: accDataBefore.CodeHash,
		}
	}

	accData := getAccData(accDataBefore)
	if accData != nil {
		out.AccountUpdate[1] = &types.StateAccountL2{
			Nonce:    int(accData.Nonce),
			Balance:  types.HexInt{Int: big.NewInt(0).Set(accData.Balance)},
			CodeHash: accData.CodeHash,
		}
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
			out.AccountKey = nAfter.Entry[0][:]
		}
		//now accountKey must has been filled
	}

	return out, nil
}

//buildSStore would return nil for a non-op (i.e. SSTORE a value identify to before state)
func (w *smtProofWriter) buildSStore(l *types.StructLogRes) (*types.StateTrace, error) {

	storeAddr := hexutil.MustDecodeBig((*l.Stack)[len(*l.Stack)-1])

	log.Debug("build SSTORE", "pc", l.Pc)

	statePath := [2]*types.SMTPath{{}, {}}
	stateUpdate := [2]*types.StateStorageL2{}

	var storageBeforeProof, storageAfterProof proofList
	var err error
	if storageBeforeProof, err = proofListFromString(mustGetStorageProof(l, posSSTOREBefore)); err != nil {
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

	if storageAfterProof, err = proofListFromString(mustGetStorageProof(l, posSSTOREAfter)); err != nil {
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

	out, err := w.traceAccountUpdate(&w.currentContract,
		func(acc *types.StateAccount) *types.StateAccount {
			//sanity check
			if accRootFromState := smt.ReverseByteOrder(statePath[0].Root); !bytes.Equal(acc.Root[:], accRootFromState) {
				panic(fmt.Errorf("unexpected storage root before: [%s] vs [%x]", acc.Root, accRootFromState))
			}
			return &types.StateAccount{
				Nonce:    acc.Nonce,
				Balance:  acc.Balance,
				CodeHash: acc.CodeHash,
				Root:     common.BytesToHash(smt.ReverseByteOrder(statePath[1].Root)),
			}
		})
	if err != nil {
		return nil, fmt.Errorf("update account %s in SSTORE fail: %s", w.currentContract, err)
	}

	out.StateKey = sAfter.Entry[0][:]
	out.StatePath = statePath
	out.StateUpdate = stateUpdate
	return out, nil
}

func (w *smtProofWriter) buildCreate(l *types.StructLogRes) (*types.StateTrace, error) {
	w.tracingAccounts[getAccountProof(l, posCREATE).Address.String()] = nil
	return w.buildCreateOrCall(l, posCREATE, posCREATE)
}

func (w *smtProofWriter) buildCreateOrCall(l *types.StructLogRes, posBefore, posAfter int) (*types.StateTrace, error) {

	proof := getAccountProof(l, posBefore)
	if proof == nil {
		return nil, fmt.Errorf("unexpected storage data for %s log at %d", l.Op, l.Pc)
	}

	proofList, err := proofListFromString(proof.Proof)
	if err != nil {
		return nil, fmt.Errorf("parse prooflist failure: %s", err)
	}

	decodeProofForAccounts(proofList, w.underlayerDb, w.tracingAccounts)

	addr, accData := getAccountDataFromProof(l, posAfter)
	if accData == nil {
		return nil, fmt.Errorf("unexpected data format for log %s", l.Op)
	}

	out, err := w.traceAccountUpdate(&addr, func(accBefore *types.StateAccount) *types.StateAccount {
		if accBefore != nil {
			accData.Root = accBefore.Root
		}
		return accData
	})
	if err != nil {
		return nil, fmt.Errorf("update account %s for creation fail: %s", addr, err)
	}
	out.CommonStateRoot = smt.ReverseByteOrder(accData.Root[:])

	return out, nil
}

// Fill smtproof field for execResult
func (w *smtProofWriter) handleLogs(logs []types.StructLogRes) error {
	logStack := []*types.StructLogRes{nil}
	contractStack := map[int]common.Address{}
	skipDepth := 0
	callEnterAddress := w.currentContract

	// now trace every OP which could cause changes on state:
	for i, sLog := range logs {

		//trace log stack by depth rather than scanning specified op
		if sl := len(logStack); sl < sLog.Depth {
			logStack = append(logStack, &logs[i])
			//update currentContract according to previous op
			contractStack[sl] = w.currentContract
			w.currentContract = callEnterAddress
		} else if sl > sLog.Depth {
			logStack = logStack[:sl-1]
			w.currentContract = contractStack[sLog.Depth]
			//reentry the last log which "cause" the calling, some handling may needed
			if err := w.handleCallEnd(logStack[len(logStack)-1]); err != nil {
				return fmt.Errorf("handle callstack popping fail: %s", err)
			}

		} else {
			logStack[sl-1] = &logs[i]
		}
		//sanity check
		if len(logStack) != sLog.Depth {
			panic("tracking log stack failure")
		}
		callEnterAddress = w.currentContract

		if skipDepth != 0 {
			if skipDepth < sLog.Depth {
				continue
			} else {
				skipDepth = 0
			}
		}

		if exD := sLog.ExtraData; exD != nil && exD.CallFailed {
			//mark current op and next ops with more depth skippable
			skipDepth = sLog.Depth
			continue
		}

		switch sLog.Op {
		case "CREATE", "CREATE2":
			if t, err := w.buildCreate(&sLog); err == nil {
				t.Index = i
				w.outTrace = append(w.outTrace, t)
			} else {
				return fmt.Errorf("handle %s log fail: %s", sLog.Op, err)
			}
			//update contract to CREATE addr
			callEnterAddress, _ = getAccountDataFromProof(&sLog, posCREATE)
		case "CALL", "CALLCODE":
			pos := posCALL
			if t, err := w.buildCreateOrCall(&sLog, pos, pos+1); err == nil {
				t.Index = i
				w.outTrace = append(w.outTrace, t)
			} else {
				return fmt.Errorf("handle %s log fail: %s", sLog.Op, err)
			}
			callEnterAddress, _ = getAccountDataFromProof(&sLog, pos)
		case "STATICCALL":
			//static call has no update on target address
			callEnterAddress, _ = getAccountDataFromProof(&sLog, posSTATICCALL)
		case "SSTORE":
			if sLog.ExtraData == nil {
				log.Warn("no storage data for SSTORE")
				break
			} else if l := len(sLog.ExtraData.ProofList); l < 2 {
				log.Warn("wrong data for SSTORE", "prooflist", l)
				break
			}

			if t, err := w.buildSStore(&sLog); err == nil {
				if t != nil {
					t.Index = i
					// sanity check
					keyRec, _ := hexutil.Decode(sLog.ExtraData.ProofList[0].Storage.Key)
					if !bytes.Equal(keyRec, t.StateUpdate[1].Key) {
						panic(fmt.Errorf("SSTORE do not have proof corresponding to its record, want %x but has %x", keyRec, []byte(t.StateUpdate[1].Key)))
					}
					w.outTrace = append(w.outTrace, t)
				} else {
					log.Debug("skip non-op SSTORE", "pc", sLog.Pc)
				}
			} else {
				return fmt.Errorf("handle SSTORE log fail: %s", err)
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

	//notice we need to init traced account status first for creation
	//notice decoding ToProof may also insert account data with the same address
	//(in the case of created on the same address)
	w.tracingAccounts[w.currentContract.String()] = nil

	accData, err := types.UnmarshalStateAccount(buf)
	if err != nil {
		return fmt.Errorf("unmarshall created acc fail: %s", err)
	}

	out, err := w.traceAccountUpdate(&w.currentContract, func(_ *types.StateAccount) *types.StateAccount { return accData })
	if err != nil {
		return fmt.Errorf("update account %s for creation fail: %s", w.currentContract, err)
	}

	out.Index = -1
	out.CommonStateRoot = smt.ReverseByteOrder(accData.Root[:])
	w.outTrace = append(w.outTrace, out)

	return nil
}

//finally update account status which is not traced in logs (Nonce added, gasBuy, gasRefund etc)
func (w *smtProofWriter) handleCallEnd(calledLog *types.StructLogRes) error {

	//no need to handle fail calling
	if calledLog.ExtraData != nil && calledLog.ExtraData.CallFailed {
		return nil
	}

	switch calledLog.Op {
	case "CREATE", "CREATE2":
		//addr, accDataBefore := getAccountDataFromProof(calledLog, posCALLBefore)
		addr, accData := getAccountDataFromProof(calledLog, posCREATEAfter)
		if accData == nil {
			return fmt.Errorf("unexpected data format for log %s", calledLog.Op)
		}

		out, err := w.traceAccountUpdate(&addr, func(accDataBefore *types.StateAccount) *types.StateAccount {
			//pick root from before state
			accData.Root = accDataBefore.Root
			return accData
		})
		if err != nil {
			return fmt.Errorf("update account for %s (after CREATE) fail: %s", addr, err)
		}
		out.Index = -1
		out.CommonStateRoot = smt.ReverseByteOrder(accData.Root[:])
		w.outTrace = append(w.outTrace, out)
	}

	return nil
}

//finally update account status which is not traced in logs (Nonce added, gasBuy, gasRefund etc)
func (w *smtProofWriter) handlePostTx(accs map[string]hexutil.Bytes) error {

	for acc, buf := range accs {

		accData, err := types.UnmarshalStateAccount(buf)
		if err != nil {
			return fmt.Errorf("unmarshall acc fail: %s", err)
		}

		addrBytes, _ := hexutil.Decode(acc)
		addr := common.BytesToAddress(addrBytes)

		out, err := w.traceAccountUpdate(&addr, func(accDataBefore *types.StateAccount) *types.StateAccount {

			//hBefore, _ := accDataBefore.Hash()
			//hAfter, _ := accData.Hash()
			//log.Info("post tx", "adr", addr.String(), "before", hBefore.Text(16), "after", hAfter.Text(16))

			//sanity check
			if !bytes.Equal(accData.Root[:], accDataBefore.Root[:]) {
				panic(fmt.Errorf("accout %s is not cleaned for state: %x vs %x", acc, accData.Root[:], accDataBefore.Root[:]))
				//log.Error("not clean failure", "error", fmt.Errorf("accout %s is not cleaned for state: %x vs %x", acc, accData.Root[:], accDataBefore.Root[:]))
			}
			return accData
		})
		if err != nil {
			return fmt.Errorf("update account %s fail: %s", addr, err)
		}

		out.Index = -1
		out.CommonStateRoot = smt.ReverseByteOrder(accData.Root[:])
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
