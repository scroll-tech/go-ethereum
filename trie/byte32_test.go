package trie

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewByte32(t *testing.T) {
	var tests = []struct {
		input               []byte
		expected            []byte
		expectedPaddingZero []byte
		expectedHash        string
		expectedHashPadding string
	}{
		{bytes.Repeat([]byte{1}, 4),
			[]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1},
			[]byte{1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			"1120169262217660912395665138727312015286293827539936259020934722663991619468",
			"11815021958450380571374861379539732018094133931187815125213818828376493710327",
		},
		{bytes.Repeat([]byte{1}, 34),
			[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
			"2219239698457798269997113163039475489501011181643161136091371987815450431154",
			"2219239698457798269997113163039475489501011181643161136091371987815450431154",
		},
	}

	for _, tt := range tests {
		byte32Result := NewByte32FromBytes(tt.input)
		byte32PaddingResult := NewByte32FromBytesPaddingZero(tt.input)
		assert.Equal(t, tt.expected, byte32Result.Bytes())
		assert.Equal(t, tt.expectedPaddingZero, byte32PaddingResult.Bytes())
		hashResult, err := byte32Result.Hash()
		assert.NoError(t, err)
		hashPaddingResult, err := byte32PaddingResult.Hash()
		assert.NoError(t, err)
		assert.Equal(t, tt.expectedHash, hashResult.String())
		assert.Equal(t, tt.expectedHashPadding, hashPaddingResult.String())
	}
}
