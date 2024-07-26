package missing_header_fields

import (
	"crypto/sha256"
	"reflect"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
)

type SHA256Checksum [sha256.Size]byte

func SHA256ChecksumFromHex(s string) SHA256Checksum {
	return SHA256Checksum(common.FromHex(s))
}

// UnmarshalJSON parses a hash in hex syntax.
func (s *SHA256Checksum) UnmarshalJSON(input []byte) error {
	return hexutil.UnmarshalFixedJSON(reflect.TypeOf(SHA256Checksum{}), input, s[:])
}

// MarshalText returns the hex representation of a.
func (s *SHA256Checksum) MarshalText() ([]byte, error) {
	return hexutil.Bytes(s[:]).MarshalText()
}
