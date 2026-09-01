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

package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/scroll-tech/go-ethereum/rpc"
)

// debugTestAPI stands in for the debug namespace: one method we want to publish
// and two we do not.
type debugTestAPI struct{}

func (*debugTestAPI) ExecutionWitness() (string, error) { return "witness", nil }
func (*debugTestAPI) SetHead() (string, error)          { return "head", nil }
func (*debugTestAPI) WriteMemProfile() (string, error)  { return "profile", nil }

type ethTestAPI struct{}

func (*ethTestAPI) BlockNumber() (string, error) { return "0x1", nil }

func testAPIs() []rpc.API {
	return []rpc.API{
		{Namespace: "debug", Version: "1.0", Service: new(debugTestAPI)},
		{Namespace: "eth", Version: "1.0", Service: new(ethTestAPI)},
	}
}

// call posts a single JSON-RPC request and reports the error code, or 0 on
// success.
func call(t *testing.T, url, method string) int {
	t.Helper()
	body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q,"params":[]}`, method)
	res, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var reply struct {
		Error *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&reply); err != nil {
		t.Fatal(err)
	}
	if reply.Error == nil {
		return 0
	}
	return reply.Error.Code
}

// TestRegisterApisMethodEntries checks the --http.api form used in production:
// a bare namespace alongside a single published debug method. The debug
// namespace must still be registered, or the published method could not be
// served at all, while its siblings must be refused.
func TestRegisterApisMethodEntries(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	modules := []string{"eth", "debug:executionWitness"}
	if err := RegisterApis(testAPIs(), modules, srv, false); err != nil {
		t.Fatalf("RegisterApis: %v", err)
	}
	httpsrv := httptest.NewServer(srv)
	t.Cleanup(httpsrv.Close)

	for _, method := range []string{"debug_executionWitness", "eth_blockNumber"} {
		if code := call(t, httpsrv.URL, method); code != 0 {
			t.Errorf("%s: error code %d, want it to be served", method, code)
		}
	}
	for _, method := range []string{"debug_setHead", "debug_writeMemProfile"} {
		if code := call(t, httpsrv.URL, method); code != -32601 {
			t.Errorf("%s: error code %d, want -32601", method, code)
		}
	}
}

// TestRegisterApisRejectsAmbiguous pins that a contradictory API list stops the
// node rather than resolving the ambiguity silently.
func TestRegisterApisRejectsAmbiguous(t *testing.T) {
	srv := rpc.NewServer()
	modules := []string{"debug", "debug:executionWitness"}
	err := RegisterApis(testAPIs(), modules, srv, false)
	if err == nil {
		t.Fatal("expected an error for a namespace listed both bare and per-method")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("error = %q, want it to mention ambiguity", err)
	}
}

// TestRegisterApisMalformedEntry pins that a malformed entry is rejected rather
// than being registered as a namespace of that literal name.
func TestRegisterApisMalformedEntry(t *testing.T) {
	for _, entry := range []string{"debug:", ":executionWitness", "debug:a:b"} {
		srv := rpc.NewServer()
		if err := RegisterApis(testAPIs(), []string{entry}, srv, false); err == nil {
			t.Errorf("entry %q was accepted, want an error", entry)
		}
	}
}

// TestRegisterApisExposeAllIgnoresFilter pins that IPC, which registers
// everything, is unaffected by method entries.
func TestRegisterApisExposeAll(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	if err := RegisterApis(testAPIs(), nil, srv, true); err != nil {
		t.Fatalf("RegisterApis: %v", err)
	}
	httpsrv := httptest.NewServer(srv)
	t.Cleanup(httpsrv.Close)
	for _, method := range []string{"debug_setHead", "debug_writeMemProfile", "eth_blockNumber"} {
		if code := call(t, httpsrv.URL, method); code == -32601 {
			t.Errorf("%s is not served under exposeAll", method)
		}
	}
}

// TestRegisterApisEmptyEntryExposesNothing pins the difference between "no
// modules configured", which falls back to the public APIs, and "modules
// configured but empty", which must expose nothing. admin_startHTTP builds the
// latter from an empty --http.api string (node/api.go).
func TestRegisterApisEmptyEntryExposesNothing(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	// Public APIs are what the fallback would register, so they are what makes
	// the difference observable.
	apis := []rpc.API{
		{Namespace: "eth", Version: "1.0", Service: new(ethTestAPI), Public: true},
		{Namespace: "debug", Version: "1.0", Service: new(debugTestAPI), Public: true},
	}
	if err := RegisterApis(apis, []string{""}, srv, false); err != nil {
		t.Fatalf("RegisterApis: %v", err)
	}
	httpsrv := httptest.NewServer(srv)
	t.Cleanup(httpsrv.Close)

	for _, method := range []string{"eth_blockNumber", "debug_setHead"} {
		if code := call(t, httpsrv.URL, method); code != -32601 {
			t.Errorf("%s: code %d, want -32601; an empty API list must expose nothing", method, code)
		}
	}
}

// TestRegisterApisNoModulesExposesPublic pins the other half: with no entries at
// all, the public APIs are served, as before.
func TestRegisterApisNoModulesExposesPublic(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	apis := []rpc.API{{Namespace: "eth", Version: "1.0", Service: new(ethTestAPI), Public: true}}
	if err := RegisterApis(apis, nil, srv, false); err != nil {
		t.Fatalf("RegisterApis: %v", err)
	}
	httpsrv := httptest.NewServer(srv)
	t.Cleanup(httpsrv.Close)

	if code := call(t, httpsrv.URL, "eth_blockNumber"); code == -32601 {
		t.Error("eth_blockNumber not served with no modules configured")
	}
}
