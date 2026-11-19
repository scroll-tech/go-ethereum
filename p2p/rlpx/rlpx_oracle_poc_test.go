// Copyright 2026 The go-ethereum Authors
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

package rlpx

import (
	"crypto/ecdsa"
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/crypto/ecies"
)

// TestHandshakeECIESInvalidCurveOracle verifies that ECIES decryption rejects
// ciphertexts containing an invalid-curve ephemeral public key with
// ErrInvalidPublicKey, not a MAC error. A MAC error would indicate the ECDH
// succeeded with the invalid point, enabling an oracle attack to extract
// bits of the receiver's private key (CVE-2026-26315).
func TestHandshakeECIESInvalidCurveOracle(t *testing.T) {
	// Generate receiver key (the node under attack).
	receiverKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	receiverECIES := ecies.ImportECDSAPublic(&receiverKey.PublicKey)

	// Create a valid ECIES ciphertext.
	plaintext := []byte("hello handshake")
	ct, err := ecies.Encrypt(rand.Reader, receiverECIES, plaintext, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// The ciphertext starts with an uncompressed EC point (0x04 || X || Y),
	// each coordinate is 32 bytes for secp256k1.
	curve := crypto.S256()
	byteLen := (curve.Params().BitSize + 7) / 8 // 32

	// Tamper: replace the ephemeral public key with a point NOT on secp256k1.
	// Use coordinates that satisfy y² != x³ + 7 (mod P).
	invalidX := new(big.Int).SetInt64(1)
	invalidY := new(big.Int).SetInt64(1)

	// Sanity: confirm the point is NOT on the curve.
	if curve.IsOnCurve(invalidX, invalidY) {
		t.Fatal("expected (1,1) to be off the secp256k1 curve")
	}

	// Build the tampered ciphertext.
	tampered := make([]byte, len(ct))
	copy(tampered, ct)
	tampered[0] = 0x04 // uncompressed point prefix
	xBytes := invalidX.Bytes()
	yBytes := invalidY.Bytes()
	// Zero-pad and copy X.
	copy(tampered[1+byteLen-len(xBytes):1+byteLen], xBytes)
	// Zero the leading bytes.
	for i := 1; i < 1+byteLen-len(xBytes); i++ {
		tampered[i] = 0
	}
	// Zero-pad and copy Y.
	copy(tampered[1+2*byteLen-len(yBytes):1+2*byteLen], yBytes)
	for i := 1 + byteLen; i < 1+2*byteLen-len(yBytes); i++ {
		tampered[i] = 0
	}

	// Decrypt with the tampered ciphertext.
	receiverECIESPriv := ecies.ImportECDSA(receiverKey)
	_, err = receiverECIESPriv.Decrypt(tampered, nil, nil)
	if err == nil {
		t.Fatal("expected decryption to fail with invalid-curve point")
	}
	if err != ecies.ErrInvalidPublicKey {
		t.Fatalf("expected ErrInvalidPublicKey, got: %v", err)
	}
}

// TestHandshakeECIESOutOfRangeCoordinates verifies that coordinates >= P are
// rejected by IsOnCurve, preventing a crash in the C secp256k1 library
// (CVE-2026-26314).
func TestHandshakeECIESOutOfRangeCoordinates(t *testing.T) {
	curve := crypto.S256()
	p := curve.Params().P

	// A point on the curve.
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if !curve.IsOnCurve(key.PublicKey.X, key.PublicKey.Y) {
		t.Fatal("generated key not on curve")
	}

	// Shift X by +P: mathematically equivalent mod P, but should be rejected
	// because the coordinate is out of the valid range [0, P).
	outOfRangeX := new(big.Int).Add(key.PublicKey.X, p)
	if curve.IsOnCurve(outOfRangeX, key.PublicKey.Y) {
		t.Fatal("IsOnCurve should reject x >= P")
	}

	// Same for Y.
	outOfRangeY := new(big.Int).Add(key.PublicKey.Y, p)
	if curve.IsOnCurve(key.PublicKey.X, outOfRangeY) {
		t.Fatal("IsOnCurve should reject y >= P")
	}

	// Negative coordinates.
	if curve.IsOnCurve(big.NewInt(-1), key.PublicKey.Y) {
		t.Fatal("IsOnCurve should reject negative x")
	}
}

// TestECIESGenerateSharedNilCoordinates verifies GenerateShared rejects a
// public key with nil coordinates.
func TestECIESGenerateSharedNilCoordinates(t *testing.T) {
	key, err := crypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	prv := ecies.ImportECDSA(key)

	badPub := &ecies.PublicKey{
		Curve: crypto.S256(),
	}
	_, err = prv.GenerateShared(badPub, 16, 16)
	if err != ecies.ErrInvalidPublicKey {
		t.Fatalf("expected ErrInvalidPublicKey for nil coords, got: %v", err)
	}
}

// importPublicKeyForTest is a helper to convert public key bytes.
func importPublicKeyForTest(pubKeyBytes []byte) (*ecies.PublicKey, error) {
	pub, err := crypto.UnmarshalPubkey(append([]byte{0x04}, pubKeyBytes...))
	if err != nil {
		return nil, err
	}
	return ecies.ImportECDSAPublic(pub), nil
}

// TestImportPublicKeyInvalidCurvePoint verifies that importPublicKey rejects
// points not on the curve via the underlying UnmarshalPubkey path.
func TestImportPublicKeyInvalidCurvePoint(t *testing.T) {
	curve := crypto.S256()
	byteLen := (curve.Params().BitSize + 7) / 8

	// Build a 64-byte public key with (1, 1) — not on secp256k1.
	pubBytes := make([]byte, 2*byteLen)
	pubBytes[byteLen-1] = 1   // X = 1
	pubBytes[2*byteLen-1] = 1 // Y = 1

	_, err := importPublicKeyForTest(pubBytes)
	if err == nil {
		t.Fatal("expected importPublicKey to reject invalid curve point")
	}
}
