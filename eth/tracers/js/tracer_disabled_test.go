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

package js

import (
	"errors"
	"testing"

	tracers2 "github.com/scroll-tech/go-ethereum/eth/tracers"
	"github.com/scroll-tech/go-ethereum/eth/tracers/internal/jsgate"
)

// TestNewJsTracerDisabled checks that the engine refuses to build a duktape VM
// while JavaScript tracers are off, even when called directly rather than
// through tracers.New.
func TestNewJsTracerDisabled(t *testing.T) {
	if tracers2.JSTracersDisabled() {
		t.Skip("JavaScript tracers already disabled by another test")
	}
	tracers2.DisableJSTracers()
	t.Cleanup(jsgate.ResetForTest)

	for _, code := range []string{
		"bigramTracer", // a bundled asset
		"{step: function(){}, fault: function(){}, result: function(){return 1}}", // user-supplied
	} {
		if _, err := newJsTracer(code, new(tracers2.Context)); !errors.Is(err, tracers2.ErrJSTracersDisabled) {
			t.Errorf("newJsTracer(%.20q) err = %v, want ErrJSTracersDisabled", code, err)
		}
	}
}
