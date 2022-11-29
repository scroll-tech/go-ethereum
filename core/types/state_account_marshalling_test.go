package types

import (
	"math/big"
	"testing"

	zktrie "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
)

func TestAccountMarshalling(t *testing.T) {
	//ensure the hash scheme consistent with designation
	example1 := &StateAccount{
		Nonce:          5,
		Balance:        big.NewInt(0).SetBytes(common.Hex2Bytes("01fffffffffffffffffffffffffffffffffffffffffffffffff9c8672c6bc7b3")),
		CodeHash:       common.Hex2Bytes("2098f5fb9e239eab3ceac3f27b81e481dc3124d55ffed523a839ee8446b64864"),
		KeccakCodeHash: common.Hex2Bytes("c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"),
		CodeSize:       0,
	}

	example2 := &StateAccount{
		Nonce:          2,
		Balance:        big.NewInt(0),
		CodeHash:       common.Hex2Bytes("28ec09723b285e17caabc4a8d52dbd097feddf408aee115cbb57c3c9c814d2b2"),
		Root:           common.HexToHash("22fb59aa5410ed465267023713ab42554c250f394901455a3366e223d5f7d147"),
		KeccakCodeHash: common.Hex2Bytes("089bfd332dfa6117cbc20756f31801ce4f5a175eb258e46bf8123317da54cd96"),
		CodeSize:       256,
	}

	for i, example := range []*StateAccount{example1, example2} {
		fields, flag := example.MarshalFields()

		h1, err := zktrie.PreHandlingElems(flag, fields)
		if err != nil {
			t.Fatal(err)
		}

		h2, err := example.Hash()
		if err != nil {
			t.Fatal(err)
		}

		if h1.BigInt().Cmp(h2) != 0 {
			t.Errorf("hash <%d> unmatched, expected [%x], get [%x]", i, h2.Bytes(), h1.Bytes())
		}
	}

}
