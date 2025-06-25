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
	penaltyThreshold := new(big.Int).SetInt64(6_000_000_000) // 6 * PRECISION
	penaltyFactor := new(big.Int).SetInt64(2_000_000_000)    // 2 * PRECISION (200% penalty)

	// Test case 1: No penalty (compression ratio >= threshold)
	t.Run("no penalty case", func(t *testing.T) {
		data := make([]byte, 100) // txSize = 100

		// Since compression ratio will be >= penaltyThreshold, penalty = 1 * PRECISION
		// feePerByte = execScalar * l1BaseFee + blobScalar * l1BlobBaseFee = 10 * 1_000_000_000 + 20 * 1_000_000_000 = 30_000_000_000
		// l1DataFee = feePerByte * txSize * penalty / PRECISION / PRECISION
		//           = 30_000_000_000 * 100 * 1_000_000_000 / 1_000_000_000 / 1_000_000_000 = 3000

		expected := new(big.Int).SetInt64(3000)

		actual := calculateEncodedL1DataFeeFeynman(
			data,
			l1BaseFee,
			l1BlobBaseFee,
			execScalar,
			blobScalar,
			penaltyThreshold,
			penaltyFactor,
		)

		assert.Equal(t, expected, actual)
	})

	// Test case 2: With penalty (compression ratio < threshold)
	t.Run("with penalty case", func(t *testing.T) {
		data := make([]byte, 100) // txSize = 100

		// Set a high penalty threshold to force penalty application
		highPenaltyThreshold := new(big.Int).SetInt64(1000_000_000_000) // 1000 * PRECISION

		// feePerByte = execScalar * l1BaseFee + blobScalar * l1BlobBaseFee = 30_000_000_000
		// l1DataFee = feePerByte * txSize * penaltyFactor / PRECISION / PRECISION
		//           = 30_000_000_000 * 100 * 2_000_000_000 / 1_000_000_000 / 1_000_000_000 = 6000

		expected := new(big.Int).SetInt64(6000)

		actual := calculateEncodedL1DataFeeFeynman(
			data,
			l1BaseFee,
			l1BlobBaseFee,
			execScalar,
			blobScalar,
			highPenaltyThreshold,
			penaltyFactor,
		)

		assert.Equal(t, expected, actual)
	})
}
