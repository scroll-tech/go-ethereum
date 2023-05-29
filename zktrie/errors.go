// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package zktrie

import (
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
)

var (
	InvalidUpdateKindError              = errors.New("invalid trie update kind, expect 'account' or 'storage'")
	InvalidStateAccountRLPEncodingError = errors.New("invalid account rlp encoding")
)

// MissingNodeError is returned by the trie functions (TryGet, TryUpdate, TryDelete)
// in the case where a trie node is not present in the local database. It contains
// information necessary for retrieving the missing node.
type MissingNodeError struct {
	NodeHash common.Hash // hash of the missing node
	Path     []byte      // hex-encoded path to the missing node
}

func (err *MissingNodeError) Error() string {
	return fmt.Sprintf("missing zktrie node %x (path %x)", err.NodeHash, err.Path)
}

type InvalidKeyLengthError struct {
	Key    []byte
	Expect int
}

func (err *InvalidKeyLengthError) Error() string {
	return fmt.Sprintf("invalid key length, expect %d, got %d, key: [%q]", err.Expect, len(err.Key), err.Key)
}

func CheckKeyLength(key []byte, expect int) error {
	if len(key) != expect {
		return &InvalidKeyLengthError{Key: key, Expect: expect}
	}
	return nil
}
