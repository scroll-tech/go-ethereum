package poseidon

import (
	"fmt"
	"testing"
)

func TestPoseidonCodeHash(t *testing.T) {
	// nil
	got := fmt.Sprintf("%s", CodeHash(nil))
	want := "0x2098f5fb9e239eab3ceac3f27b81e481dc3124d55ffed523a839ee8446b64864"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}

	// single byte
	got = fmt.Sprintf("%s", CodeHash([]byte{0}))
	want = "0x0ee069e6aa796ef0e46cbd51d10468393d443a00f5affe72898d9ab62e335e16"

	if got != want {
		t.Errorf("got %q, wanted %q", got, want)
	}
}
