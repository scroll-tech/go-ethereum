// Copyright 2021 The go-ethereum Authors
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

package types

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/iden3/go-iden3-crypto/utils"

	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
)

var (
	ErrInvalidLength = errors.New("StateAccount: invalid input length")
)

// MarshalFields, the bytes scheme is:
// [0:32] Nonce uint64 big-endian in 32 byte
// [32:64] Balance
// [64:96] KeccakCodeHash
// [96:128] Root
// [128:160] PoseidonCodeHash
// [160:192] CodeSize
func (s *StateAccount) MarshalFields() ([]zkt.Byte32, uint32) {
	fields := make([]zkt.Byte32, 6)

	if !utils.CheckBigIntInField(s.Balance) {
		panic("StateAccount balance overflow")
	}

	binary.BigEndian.PutUint64(fields[0][24:], s.Nonce) // 8-byte value in a 32-byte field
	s.Balance.FillBytes(fields[1][:])
	copy(fields[2][:], s.KeccakCodeHash)
	copy(fields[3][:], s.Root.Bytes())
	copy(fields[4][:], s.PoseidonCodeHash)
	binary.BigEndian.PutUint64(fields[5][24:], s.CodeSize) // 8-byte value in a 32-byte field

	// The returned flag shows which items cannot be encoded as a field elements.
	//
	// +-------+---------+--------+--------+----------+----------+
	// | nonce | balance | keccak |  root  | poseidon | codesize |
	// +-------+---------+--------+--------+----------+----------+
	//     0        0        1        0         0          0

	flag := uint32(4)

	return fields, flag
}

func UnmarshalStateAccount(bytes []byte) (*StateAccount, error) {
	if len(bytes) != 192 {
		return nil, ErrInvalidLength
	}
	acc := new(StateAccount)
	acc.Nonce = binary.BigEndian.Uint64(bytes[24:])
	acc.Balance = new(big.Int).SetBytes(bytes[32:64])
	acc.KeccakCodeHash = make([]byte, 32)
	copy(acc.KeccakCodeHash, bytes[64:96])
	acc.Root = common.Hash{}
	acc.Root.SetBytes(bytes[96:128])
	acc.PoseidonCodeHash = make([]byte, 32)
	copy(acc.PoseidonCodeHash, bytes[128:160])
	acc.CodeSize = binary.BigEndian.Uint64(bytes[(160 + 24):])

	return acc, nil
}
