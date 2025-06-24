package fees

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestL1DataFeeBeforeCurie(t *testing.T) {
	l1BaseFee := new(big.Int).SetUint64(15000000)
	overhead := new(big.Int).SetUint64(100)
	scalar := new(big.Int).SetUint64(10)

	data := []byte{0, 10, 1, 0}

	expected := new(big.Int).SetUint64(30) // 30.6
	actual := calculateEncodedL1DataFee(data, overhead, l1BaseFee, scalar)
	assert.Equal(t, expected, actual)
}

func TestL1DataFeeAfterCurie(t *testing.T) {
	l1BaseFee := new(big.Int).SetUint64(1500000000)
	l1BlobBaseFee := new(big.Int).SetUint64(150000000)
	commitScalar := new(big.Int).SetUint64(10)
	blobScalar := new(big.Int).SetUint64(10)

	data := []byte{0, 10, 1, 0}

	expected := new(big.Int).SetUint64(21)
	actual := calculateEncodedL1DataFeeCurie(data, l1BaseFee, l1BlobBaseFee, commitScalar, blobScalar)
	assert.Equal(t, expected, actual)
}
func TestL1DataFeeFeynman(t *testing.T) {
	l1BaseFee := new(big.Int).SetInt64(1_000_000_000)
	l1BlobBaseFee := new(big.Int).SetInt64(1_000_000_000)
	execScalar := new(big.Int).SetInt64(10)
	blobScalar := new(big.Int).SetInt64(20)
	data := make([]byte, 10) // txSize = 10

	// feePerByte = execScalar * l1BaseFee + blobScalar * l1BlobBaseFee = 10_000_000_000 + 20_000_000_000 = 30_000_000_000
	// l1DataFee = precision * txSize * feePerByte / precision / precision
	//           = 1_000_000_000 * 10 * 30_000_000_000 / 1_000_000_000 / 1_000_000 = 300_000_000_000_000_000 / 1_000_000_000 / 1_000_000_000 = 300
	//TODO for now compression_ratio = precision, placeholder

	expected := new(big.Int).SetInt64(300)

	actual := calculateEncodedL1DataFeeFeynman(
		data,
		l1BaseFee,
		l1BlobBaseFee,
		execScalar,
		blobScalar,
	)

	assert.Equal(t, expected, actual)
}
