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

package rpc

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseAPIEntries(t *testing.T) {
	tests := []struct {
		name       string
		entries    []string
		namespaces []string
		filtered   bool
		wantErr    string
	}{
		{
			name:       "bare namespaces only leave dispatch unfiltered",
			entries:    []string{"eth", "scroll", "net", "web3"},
			namespaces: []string{"eth", "scroll", "net", "web3"},
			filtered:   false,
		},
		{
			name:       "method entry registers its namespace and installs a filter",
			entries:    []string{"eth", "debug:executionWitness"},
			namespaces: []string{"eth", "debug"},
			filtered:   true,
		},
		{
			name:       "several methods of one namespace",
			entries:    []string{"debug:executionWitness", "debug:traceCall"},
			namespaces: []string{"debug"},
			filtered:   true,
		},
		{
			name:       "surrounding whitespace is ignored",
			entries:    []string{" eth ", " debug : executionWitness "},
			namespaces: []string{"eth", "debug"},
			filtered:   true,
		},
		{
			name:       "empty entries are skipped",
			entries:    []string{"eth", "", "  "},
			namespaces: []string{"eth"},
			filtered:   false,
		},
		{
			name:    "bare and per-method for one namespace is ambiguous",
			entries: []string{"debug", "debug:executionWitness"},
			wantErr: "listed both bare and per-method",
		},
		{
			name:    "too many separators",
			entries: []string{"debug:a:b"},
			wantErr: "expected",
		},
		{
			name:    "empty method name",
			entries: []string{"debug:"},
			wantErr: "empty method name",
		},
		{
			name:    "empty namespace",
			entries: []string{":executionWitness"},
			wantErr: "empty namespace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespaces, filter, err := ParseAPIEntries(tt.entries)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(namespaces, tt.namespaces) {
				t.Errorf("namespaces = %v, want %v", namespaces, tt.namespaces)
			}
			if got := filter != nil; got != tt.filtered {
				t.Errorf("filter installed = %v, want %v", got, tt.filtered)
			}
		})
	}
}

func TestMethodFilterAllows(t *testing.T) {
	_, filter, err := ParseAPIEntries([]string{"eth", "net", "debug:executionWitness"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	tests := []struct {
		method string
		want   bool
		why    string
	}{
		{"eth_getBalance", true, "bare namespace is unrestricted"},
		{"eth_subscribe", true, "bare namespace covers subscriptions"},
		{"net_version", true, "bare namespace is unrestricted"},
		{"debug_executionWitness", true, "explicitly allowed"},
		{"debug_setHead", false, "same namespace, not listed"},
		{"debug_writeMemProfile", false, "same namespace, not listed"},
		{"debug_vmodule", false, "same namespace, not listed"},
		{"rpc_modules", false, "namespace not mentioned, denied by default"},
		{"debug_", false, "malformed, no method name"},
		{"_setHead", false, "malformed, no namespace"},
		{"nonsense", false, "not addressable as namespace_method"},
	}
	for _, tt := range tests {
		if got := filter.Allows(tt.method); got != tt.want {
			t.Errorf("Allows(%q) = %v, want %v (%s)", tt.method, got, tt.want, tt.why)
		}
	}
}

// TestMethodFilterNilAllowsAll pins the default: without a filter nothing is
// refused, so existing deployments are unaffected.
func TestMethodFilterNilAllowsAll(t *testing.T) {
	var filter *MethodFilter
	for _, method := range []string{"debug_setHead", "eth_getBalance", "anything"} {
		if !filter.Allows(method) {
			t.Errorf("nil filter refused %q", method)
		}
	}
}

func TestMethodFilterMethodNames(t *testing.T) {
	_, filter, err := ParseAPIEntries([]string{"eth", "debug:traceCall", "debug:executionWitness"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := []string{"debug_executionWitness", "debug_traceCall"}
	if got := filter.methodNames(); !reflect.DeepEqual(got, want) {
		t.Errorf("methodNames() = %v, want %v", got, want)
	}
}

// TestMethodFilterDeniesUnlistedNamespace pins the fail-closed default: a
// namespace nobody listed is refused, not waved through.
func TestMethodFilterDeniesUnlistedNamespace(t *testing.T) {
	_, filter, err := ParseAPIEntries([]string{"debug:executionWitness"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	for _, method := range []string{"eth_getBalance", "admin_peers", "personal_unlockAccount"} {
		if filter.Allows(method) {
			t.Errorf("Allows(%q) = true, want false for an unlisted namespace", method)
		}
	}
	// AllowMethod is how a Server re-opens what it registered itself, one method
	// at a time so the rest of that namespace stays denied.
	filter.AllowMethod("rpc", "modules")
	if !filter.Allows("rpc_modules") {
		t.Error("Allows(rpc_modules) = false after AllowMethod(rpc, modules)")
	}
	if filter.Allows("rpc_somethingAddedLater") {
		t.Error("Allows(rpc_somethingAddedLater) = true; allowing one method must not open the namespace")
	}
}

// TestParseAPIEntriesRejectsSubscriptions pins that a per-method subscription
// entry is refused, since the filter cannot see the subscription name.
func TestParseAPIEntriesRejectsSubscriptions(t *testing.T) {
	for _, entry := range []string{"eth:subscribe", "eth:unsubscribe"} {
		if _, _, err := ParseAPIEntries([]string{entry}); err == nil {
			t.Errorf("entry %q was accepted, want an error", entry)
		}
	}
}

// TestMethodFilterZeroValue pins that the exported type survives being
// constructed by a caller rather than by ParseAPIEntries. SetMethodFilter calls
// AllowMethod, which would otherwise write to a nil map and panic.
func TestMethodFilterZeroValue(t *testing.T) {
	srv := NewServer()
	defer srv.Stop()

	if unknown := srv.SetMethodFilter(&MethodFilter{}); len(unknown) != 0 {
		t.Errorf("SetMethodFilter reported unknown methods %v, want none", unknown)
	}
	// The metadata method the Server allows itself must still be reachable, and
	// nothing else.
	filter, _ := srv.services.filter.Load().(*MethodFilter)
	if !filter.Allows("rpc_modules") {
		t.Error("rpc_modules denied after SetMethodFilter on a zero-value filter")
	}
	if filter.Allows("eth_getBalance") {
		t.Error("eth_getBalance allowed by a zero-value filter")
	}
}
