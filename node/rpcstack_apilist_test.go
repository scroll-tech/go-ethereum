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
// a bare namespace alongside a single published debug method. The namespace
// must still be registered, while its other methods are refused.
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

// TestRegisterApisExposeAll pins the exposeAll escape hatch: every namespace is
// registered regardless of the API list. This is not the IPC path; IPC never
// calls RegisterApis.
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

// TestRegisterApisExposesNothingWithoutEntries pins that an API list naming
// nothing exposes nothing. These are the shapes the CLI and admin_startHTTP
// produce: SplitAndTrim drops empty fields, so --http.api "" arrives as nil,
// while admin_startHTTP("") arrives as [""].
func TestRegisterApisExposesNothingWithoutEntries(t *testing.T) {
	apis := []rpc.API{
		{Namespace: "eth", Version: "1.0", Service: new(ethTestAPI)},
		{Namespace: "debug", Version: "1.0", Service: new(debugTestAPI)},
	}
	for _, modules := range [][]string{nil, {""}, {" "}} {
		srv := rpc.NewServer()
		if err := RegisterApis(apis, modules, srv, false); err != nil {
			srv.Stop()
			t.Fatalf("RegisterApis(%q): %v", modules, err)
		}
		httpsrv := httptest.NewServer(srv)
		for _, method := range []string{"eth_blockNumber", "debug_setHead"} {
			if code := call(t, httpsrv.URL, method); code != -32601 {
				t.Errorf("modules=%q: %s code %d, want -32601", modules, method, code)
			}
		}
		httpsrv.Close()
		srv.Stop()
	}
}

// TestRegisterApisPaddedEntry pins that surrounding whitespace is trimmed, so a
// list written as "eth, debug" does what it reads like.
func TestRegisterApisPaddedEntry(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	if err := RegisterApis(testAPIs(), []string{" eth ", " debug : executionWitness "}, srv, false); err != nil {
		t.Fatalf("RegisterApis: %v", err)
	}
	httpsrv := httptest.NewServer(srv)
	t.Cleanup(httpsrv.Close)

	if code := call(t, httpsrv.URL, "eth_blockNumber"); code != 0 {
		t.Errorf("eth_blockNumber code %d, want it served", code)
	}
	if code := call(t, httpsrv.URL, "debug_executionWitness"); code != 0 {
		t.Errorf("debug_executionWitness code %d, want it served", code)
	}
	if code := call(t, httpsrv.URL, "debug_setHead"); code != -32601 {
		t.Errorf("debug_setHead code %d, want -32601", code)
	}
}

// TestRegisterApisUnknownMethodFails pins that a method nobody offers stops the
// node rather than being silently denied.
func TestRegisterApisUnknownMethodFails(t *testing.T) {
	srv := rpc.NewServer()
	t.Cleanup(srv.Stop)
	err := RegisterApis(testAPIs(), []string{"debug:executionWitnes"}, srv, false)
	if err == nil {
		t.Fatal("expected an error for a method the server does not offer")
	}
	if !strings.Contains(err.Error(), "unknown method") {
		t.Errorf("error = %q, want it to mention an unknown method", err)
	}
}
