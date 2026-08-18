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

package tracers

import (
	"errors"
	"testing"

	"github.com/scroll-tech/go-ethereum/eth/tracers/internal/jsgate"
)

// stubTracer is a placeholder returned by the lookups registered below. New()
// only ever returns it, so no method on it is called.
type stubTracer struct{ Tracer }

// TestDisableJSTracers verifies that disabling the interpreted engines keeps the
// native lookups reachable while the wildcard lookup is skipped entirely.
func TestDisableJSTracers(t *testing.T) {
	// Save and restore the package-level state, other tests share it.
	savedLookups, savedWildcards := lookups, wildcardLookups
	t.Cleanup(func() {
		lookups, wildcardLookups = savedLookups, savedWildcards
		jsgate.ResetForTest()
	})

	var wildcardCalled bool
	lookups, wildcardLookups = nil, nil
	jsgate.ResetForTest()

	// A native lookup that only answers to one name, like the Go tracers do.
	RegisterLookup(false, func(code string, ctx *Context) (Tracer, error) {
		if code != "callTracer" {
			return nil, errors.New("unknown tracer")
		}
		return &stubTracer{}, nil
	})
	// A wildcard lookup that accepts anything, like the JavaScript engine does.
	RegisterLookup(true, func(code string, ctx *Context) (Tracer, error) {
		wildcardCalled = true
		return &stubTracer{}, nil
	})

	// While enabled, arbitrary code reaches the wildcard lookup.
	if _, err := New("(function(){})", new(Context)); err != nil {
		t.Fatalf("custom tracer rejected while enabled: %v", err)
	}
	if !wildcardCalled {
		t.Fatal("wildcard lookup was not consulted while enabled")
	}

	DisableJSTracers()
	wildcardCalled = false

	// Named native tracers must keep working.
	if _, err := New("callTracer", new(Context)); err != nil {
		t.Fatalf("native tracer rejected while JS is disabled: %v", err)
	}
	if wildcardCalled {
		t.Fatal("wildcard lookup was consulted for a native tracer name")
	}

	// Arbitrary code must be rejected without ever reaching the engine.
	if _, err := New("(function(){})", new(Context)); !errors.Is(err, ErrJSTracersDisabled) {
		t.Fatalf("custom tracer error = %v, want %v", err, ErrJSTracersDisabled)
	}
	if wildcardCalled {
		t.Fatal("wildcard lookup was consulted while JS is disabled")
	}
	if !JSTracersDisabled() {
		t.Fatal("JSTracersDisabled() reported false after DisableJSTracers()")
	}
}
