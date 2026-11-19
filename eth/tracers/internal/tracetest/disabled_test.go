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

package tracetest

import (
	"errors"
	"testing"

	"github.com/scroll-tech/go-ethereum/eth/tracers"
	"github.com/scroll-tech/go-ethereum/eth/tracers/internal/jsgate"
)

// customJSTracer is a minimal user-supplied tracer, the shape an attacker would
// send to debug_traceCall.
const customJSTracer = `{step: function(){}, fault: function(){}, result: function(){return 1}}`

// TestJSTracersDisabled exercises the real native and JavaScript lookups
// together, which the eth/tracers unit test cannot do because both engines
// import that package.
func TestJSTracersDisabled(t *testing.T) {
	jsgate.ResetForTest()
	t.Cleanup(jsgate.ResetForTest)

	// While enabled, every category resolves.
	for _, code := range []string{"callTracer", "bigramTracer", customJSTracer} {
		if _, err := tracers.New(code, new(tracers.Context)); err != nil {
			t.Fatalf("enabled: %q rejected: %v", code, err)
		}
	}
	// An unrecognised name reports "tracer not found", not "disabled".
	if _, err := tracers.New("nonexistentTracer{", new(tracers.Context)); err == nil {
		t.Fatal("enabled: unknown tracer accepted")
	} else if errors.Is(err, tracers.ErrJSTracersDisabled) {
		t.Fatalf("enabled: unknown tracer reported as disabled: %v", err)
	}

	tracers.DisableJSTracers()

	// Every native Go tracer must keep working. Note prestateTracer is in both
	// lists upstream: the native lookup runs first, so it shadows the JavaScript
	// asset of the same name and survives.
	for _, code := range []string{
		"callTracer", "prestateTracer", "4byteTracer", "flatCallTracer", "noopTracerNative",
	} {
		if _, err := tracers.New(code, new(tracers.Context)); err != nil {
			t.Errorf("disabled: native %q rejected: %v", code, err)
		}
	}
	// User-supplied code and every JavaScript-only bundled tracer must be
	// refused, because serving them would still build a duktape VM.
	for _, code := range []string{
		customJSTracer,
		"4byteTracerLegacy", "bigramTracer", "callTracerJs", "callTracerLegacy",
		"evmdisTracer", "noopTracer", "opcountTracer", "trigramTracer", "unigramTracer",
	} {
		if _, err := tracers.New(code, new(tracers.Context)); !errors.Is(err, tracers.ErrJSTracersDisabled) {
			t.Errorf("disabled: %q err = %v, want ErrJSTracersDisabled", code, err)
		}
	}
}
