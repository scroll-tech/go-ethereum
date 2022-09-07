package poseidon

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

const defaultPoseidonChunk = 3

func CodeHash(code []byte) (h common.Hash) {
	// special case for nil hash
	if len(code) == 0 {
		return crypto.Keccak256Hash(nil)
	}

	cap := int64(len(code))

	// Step1: pad code with 0x0 (STOP) so len(code) % 16 == 0
	// Step2: for every 16 byte, convert it into Fr, so we get a Fr array
	var length = (len(code) + 15) / 16

	Frs := make([]*big.Int, length)
	ii := 0

	for ii < length-1 {
		Frs[ii] = big.NewInt(0)
		Frs[ii].SetBytes(code[ii*16 : (ii+1)*16])
		ii++
	}

	Frs[ii] = big.NewInt(0)
	bytes := make([]byte, 16)
	copy(bytes, code[ii*16:])
	Frs[ii].SetBytes(bytes)

	// Step3: pad Fr array with 0 to even length.
	// FIXME: no need, Hash would pad it with 0 automatically, and Hash
	// need actual length as a flag

	// Step4: Apply the array onto a sponge process with current poseidon scheme
	// (3 Frs permutation and 1 Fr for output, so the throughout is 2 Frs)
	hash, _ := HashWithCap(Frs, defaultPoseidonChunk, cap)

	// Step5(short term, for compatibility): convert final root Fr as u256 (big-endian represent)
	codeHash := common.Hash{}
	hash.FillBytes(codeHash[:])

	return codeHash
}
