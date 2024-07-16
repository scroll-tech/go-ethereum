package cmd

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/export-headers-toolkit/types"
)

func TestMissingHeaderWriter(t *testing.T) {
	vanity1 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}
	vanity2 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02}
	vanity8 := [32]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08}

	var expectedBytes []byte
	expectedBytes = append(expectedBytes, 0x03)
	expectedBytes = append(expectedBytes, vanity1[:]...)
	expectedBytes = append(expectedBytes, vanity2[:]...)
	expectedBytes = append(expectedBytes, vanity8[:]...)

	seenVanity := map[[32]byte]bool{
		vanity8: true,
		vanity1: true,
		vanity2: true,
	}
	var buf []byte
	bytesBuffer := bytes.NewBuffer(buf)
	mhw := newMissingHeaderWriter(bytesBuffer, seenVanity)

	mhw.writeVanities()
	assert.Equal(t, expectedBytes, bytesBuffer.Bytes())

	// Header0
	{
		seal := randomSeal(65)
		header := types.NewHeader(0, 2, append(vanity1[:], seal...))
		mhw.write(header)

		expectedBytes = append(expectedBytes, 0x00) // index 0
		expectedBytes = append(expectedBytes, 0x00) // difficulty 2, seal length 65
		expectedBytes = append(expectedBytes, seal...)
		assert.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header1
	{
		seal := randomSeal(65)
		header := types.NewHeader(1, 1, append(vanity2[:], seal...))
		mhw.write(header)

		expectedBytes = append(expectedBytes, 0x01) // index 1
		expectedBytes = append(expectedBytes, 0x01) // difficulty 1, seal length 65
		expectedBytes = append(expectedBytes, seal...)
		assert.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header2
	{
		seal := randomSeal(85)
		header := types.NewHeader(2, 2, append(vanity2[:], seal...))
		mhw.write(header)

		expectedBytes = append(expectedBytes, 0x01) // index 1
		expectedBytes = append(expectedBytes, 0x02) // difficulty 2, seal length 85
		expectedBytes = append(expectedBytes, seal...)
		assert.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header3
	{
		seal := randomSeal(85)
		header := types.NewHeader(3, 1, append(vanity8[:], seal...))
		mhw.write(header)

		expectedBytes = append(expectedBytes, 0x02) // index 2
		expectedBytes = append(expectedBytes, 0x03) // difficulty 1, seal length 85
		expectedBytes = append(expectedBytes, seal...)
		assert.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}

	// Header4
	{
		seal := randomSeal(65)
		header := types.NewHeader(4, 2, append(vanity1[:], seal...))
		mhw.write(header)

		expectedBytes = append(expectedBytes, 0x00) // index 0
		expectedBytes = append(expectedBytes, 0x00) // difficulty 2, seal length 65
		expectedBytes = append(expectedBytes, seal...)
		assert.Equal(t, expectedBytes, bytesBuffer.Bytes())
	}
}

func randomSeal(length int) []byte {
	buf := make([]byte, length)
	_, _ = rand.Read(buf)
	return buf
}
