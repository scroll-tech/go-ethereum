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

	"github.com/iden3/go-iden3-crypto/poseidon"
	"github.com/iden3/go-iden3-crypto/utils"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types/smt"
)

var (
	ErrInvalidLength = errors.New("StateAccount: invalid input length")
)

// StateAccount is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type StateAccount struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash // merkle root of the storage trie
	CodeHash []byte
}

// Hash of StateAccount
// AccountHash = Hash(
//	Hash(nonce, balance),
//  Hash(
//	  Root,
//	  Hash(codeHashFirst16, codeHashLast16)
//  ))
func (s *StateAccount) Hash() (*big.Int, error) {
	nonce := new(big.Int).SetUint64(s.Nonce)
	if s.Balance == nil {
		s.Balance = new(big.Int)
	}
	hash1, err := poseidon.Hash([]*big.Int{nonce, s.Balance})
	if err != nil {
		return nil, err
	}

	codeHashFirst16 := new(big.Int).SetBytes(s.CodeHash[0:16])
	codeHashLast16 := new(big.Int).SetBytes(s.CodeHash[16:32])
	hash2, err := poseidon.Hash([]*big.Int{codeHashFirst16, codeHashLast16})
	if err != nil {
		return nil, err
	}

	rootHash, err := smt.NewHashFromBytes(s.Root.Bytes())
	if err != nil {
		return nil, err
	}
	hash3, err := poseidon.Hash([]*big.Int{hash2, rootHash.BigInt()})
	if err != nil {
		return nil, err
	}

	hash4, err := poseidon.Hash([]*big.Int{hash1, hash3})
	if err != nil {
		return nil, err
	}
	return hash4, nil
}

// MarshalBytes
// [0:7] Nonce uint64 little-endian
// [32:64] Balance
// [64:96] Root
// [96:128] CodeHash
func (s *StateAccount) MarshalBytes() []byte {
	bytes := make([]byte, 128)
	binary.LittleEndian.PutUint64(bytes, s.Nonce)

	if !utils.CheckBigIntInField(s.Balance) {
		panic("balance overflow")
	}
	s.Balance.FillBytes(bytes[32:64])

	copy(bytes[64:96], s.Root.Bytes())
	copy(bytes[96:128], s.CodeHash)
	return bytes
}

func UnmarshalStateAccount(bytes []byte) (*StateAccount, error) {
	if len(bytes) != 128 {
		return nil, ErrInvalidLength
	}
	acc := new(StateAccount)
	acc.Nonce = binary.LittleEndian.Uint64(bytes[:8])
	acc.Balance = new(big.Int).SetBytes(bytes[32:64])
	acc.Root = common.Hash{}
	acc.Root.SetBytes(bytes[64:96])
	acc.CodeHash = make([]byte, 32)
	copy(acc.CodeHash, bytes[96:128])
	return acc, nil
}
