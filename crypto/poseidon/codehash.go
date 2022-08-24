package poseidon

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

const defaultPoseidonChunk = 3

// @todo: This is just a rough first draft, optimize it once we have test vectors
func CodeHash(code []byte) (h common.Hash) {
	// @todo: decide how to handle nil hash
	if code == nil {
		return crypto.Keccak256Hash(nil)
	}

	cap := int64(len(code))

	// Step1: pad code with 0x0 (STOP) so len(code) % 16 == 0
	if len(code)%16 != 0 {
		newLen := (len(code)/16 + 1) * 16
		code = append(code, make([]byte, newLen-len(code))...)
	}

	// Step2: for every 16 byte, convert it into Fr, so we get a Fr array
	Frs := make([]*big.Int, len(code)/16)

	for ii := 0; ii < len(code)/16; ii++ {
		Frs[ii] = big.NewInt(0)
		Frs[ii].SetBytes(code[ii*16 : (ii+1)*16])
	}

	// Step3: pad Fr array with 0 to even length.
	// FIXME: no need, Hash would pad it with 0 automatically, and Hash
	// need actual length as a flag
	/*if len(Frs)%2 == 1 {
		Frs = append(Frs, big.NewInt(0))
	}*/

	// Step4: Apply the array onto a sponge process with current poseidon scheme
	// (3 Frs permutation and 1 Fr for output, so the throughout is 2 Frs)
	// @todo
	hash, _ := HashWithCap(Frs, defaultPoseidonChunk, cap)

	// Step5(short term, for compatibility): convert final root Fr as u256 (big-endian represent)
	// @todo: confirm endianness
	codeHash := common.Hash{}
	hash.FillBytes(codeHash[:])

	return codeHash
}
