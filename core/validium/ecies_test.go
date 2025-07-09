package validium

import (
	"encoding/hex"
	"testing"
)

func TestEciesDecrypt(t *testing.T) {
	message := "127b15f37acbeaa4188a3388689445ae892787bcrandomSeed"

	// test key and ciphertext generated using eciesjs
	sk, _ := hex.DecodeString("f5cee12961c0b9b25fd406ee87383a48b275b227050481892e57e9e836f3d38e")
	ciphertext, _ := hex.DecodeString("040a8df01d04f193e175c468a034e057076a9d62dafa0917134f496b1fade97296a3936706206d5d5742849dd303fa1155df026fe59e8014ac44720085cccc0e7a8db689efa7fe30e32ed826fe4f0e4dcad865e3a133f4dd1b47b0e3e614fb4edf43bcb853e6c87ff5a5deb33ffc1c4d7d7ee8d5c253c0c445bc6076bf7403a2cbe9c5285c86a6852a6a70cb2ce6e0e38324df")

	plaintext, err := DecryptEcies(ciphertext, sk)
	if err != nil {
		t.Fatal(err)
	}

	if expected, actual := message, string(plaintext); expected != actual {
		t.Fatalf("Unexpected decryption result, expected = %s, actual = %s", expected, actual)
	}
}
