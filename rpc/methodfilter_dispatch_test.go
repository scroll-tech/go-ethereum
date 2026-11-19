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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// rpcResponse is the subset of a JSON-RPC reply the filter tests care about.
type rpcResponse struct {
	ID     int             `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// filteredTestServer builds a server exposing the "test" and "nftest" services,
// restricted by the given API list entries.
func filteredTestServer(t *testing.T, entries ...string) *Server {
	t.Helper()
	srv := newTestServer()
	_, filter, err := ParseAPIEntries(entries)
	if err != nil {
		t.Fatalf("ParseAPIEntries(%v): %v", entries, err)
	}
	srv.SetMethodFilter(filter)
	t.Cleanup(srv.Stop)
	return srv
}

// filteredHTTPServer returns the URL of an HTTP server whose dispatch is
// restricted by the given API list entries.
func filteredHTTPServer(t *testing.T, entries ...string) string {
	t.Helper()
	httpsrv := httptest.NewServer(filteredTestServer(t, entries...))
	t.Cleanup(httpsrv.Close)
	return httpsrv.URL
}

// postRaw sends a raw JSON body and decodes the reply as a single response.
func postRaw(t *testing.T, url, body string) rpcResponse {
	t.Helper()
	raw := postRawBytes(t, url, body)
	var resp rpcResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return resp
}

func postRawBytes(t *testing.T, url, body string) []byte {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(raw)
}

func call(method string, id int) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":%q,"params":[]}`, id, method)
}

// wantDenied asserts the reply is a method-not-found error naming the method.
func wantDenied(t *testing.T, resp rpcResponse, method string) {
	t.Helper()
	if resp.Error == nil {
		t.Fatalf("%s: expected an error, got result %s", method, resp.Result)
	}
	if resp.Error.Code != -32601 {
		t.Errorf("%s: error code = %d, want -32601", method, resp.Error.Code)
	}
	want := fmt.Sprintf("the method %s does not exist/is not available", method)
	if resp.Error.Message != want {
		t.Errorf("%s: message = %q, want %q", method, resp.Error.Message, want)
	}
}

func wantAllowed(t *testing.T, resp rpcResponse, method string) {
	t.Helper()
	if resp.Error != nil {
		t.Fatalf("%s: unexpected error %d %q", method, resp.Error.Code, resp.Error.Message)
	}
	if len(resp.Result) == 0 {
		t.Fatalf("%s: empty result", method)
	}
}

// TestMethodFilterHTTP covers per-method allow and deny over HTTP, and checks a
// denied method is reported exactly like one that was never registered.
func TestMethodFilterHTTP(t *testing.T) {
	url := filteredHTTPServer(t, "nftest", "test:echo")

	t.Run("allowed method of a restricted namespace", func(t *testing.T) {
		resp := postRaw(t, url, `{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",3]}`)
		wantAllowed(t, resp, "test_echo")
	})
	t.Run("denied method of a restricted namespace", func(t *testing.T) {
		wantDenied(t, postRaw(t, url, call("test_rets", 2)), "test_rets")
	})
	t.Run("bare namespace stays fully reachable", func(t *testing.T) {
		resp := postRaw(t, url, `{"jsonrpc":"2.0","id":3,"method":"nftest_echo","params":[7]}`)
		wantAllowed(t, resp, "nftest_echo")
	})
	t.Run("unknown method in a restricted namespace", func(t *testing.T) {
		wantDenied(t, postRaw(t, url, call("test_noSuchMethod", 4)), "test_noSuchMethod")
	})
	t.Run("unknown namespace", func(t *testing.T) {
		wantDenied(t, postRaw(t, url, call("nosuch_method", 5)), "nosuch_method")
	})
}

// TestMethodFilterIndistinguishable pins that a refused method and a method that
// was never registered produce byte-identical replies once the differing method
// name is accounted for.
func TestMethodFilterIndistinguishable(t *testing.T) {
	url := filteredHTTPServer(t, "test:echo")

	// test_rets exists but is refused; test_absent was never registered.
	refused := postRawBytes(t, url, call("test_rets", 1))
	absent := postRawBytes(t, url, call("test_absent", 1))

	normalised := bytes.Replace(refused, []byte("test_rets"), []byte("METHOD"), -1)
	wantShape := bytes.Replace(absent, []byte("test_absent"), []byte("METHOD"), -1)
	if !bytes.Equal(normalised, wantShape) {
		t.Errorf("refused and unregistered replies differ:\n refused = %s\n absent  = %s", refused, absent)
	}
}

// TestMethodFilterBatch checks each element of a batch is judged on its own.
func TestMethodFilterBatch(t *testing.T) {
	url := filteredHTTPServer(t, "nftest", "test:echo")

	body := `[
		{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["a",1]},
		{"jsonrpc":"2.0","id":2,"method":"test_rets","params":[]},
		{"jsonrpc":"2.0","id":3,"method":"nftest_echo","params":[2]},
		{"jsonrpc":"2.0","id":4,"method":"test_noSuchMethod","params":[]}
	]`
	raw := postRawBytes(t, url, body)
	var resps []rpcResponse
	if err := json.Unmarshal(raw, &resps); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	if len(resps) != 4 {
		t.Fatalf("got %d responses, want 4: %s", len(resps), raw)
	}
	byID := make(map[int]rpcResponse, len(resps))
	for _, r := range resps {
		byID[r.ID] = r
	}
	wantAllowed(t, byID[1], "test_echo")
	wantDenied(t, byID[2], "test_rets")
	wantAllowed(t, byID[3], "nftest_echo")
	wantDenied(t, byID[4], "test_noSuchMethod")
}

// TestMethodFilterBareNamespacesUnchanged pins that an API list without any
// method entry behaves exactly as before: no filter, nothing refused.
func TestMethodFilterBareNamespacesUnchanged(t *testing.T) {
	url := filteredHTTPServer(t, "test", "nftest")

	bodies := []string{
		`{"jsonrpc":"2.0","id":1,"method":"test_echo","params":["x",1]}`,
		call("test_rets", 2),
	}
	for _, body := range bodies {
		if resp := postRaw(t, url, body); resp.Error != nil {
			t.Errorf("refused with a bare namespace: %d %q\n request: %s", resp.Error.Code, resp.Error.Message, body)
		}
	}
}

// TestMethodFilterWebsocket checks WebSocket enforces the same allowlist as
// HTTP, since the check sits on the shared dispatch path.
func TestMethodFilterWebsocket(t *testing.T) {
	srv := filteredTestServer(t, "nftest", "test:echo")
	httpsrv := httptest.NewServer(srv.WebsocketHandler([]string{"*"}))
	t.Cleanup(httpsrv.Close)
	wsURL := "ws:" + strings.TrimPrefix(httpsrv.URL, "http:")

	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	var result echoResult
	if err := client.Call(&result, "test_echo", "x", 3); err != nil {
		t.Errorf("test_echo over ws: unexpected error %v", err)
	}
	err = client.Call(new(interface{}), "test_rets")
	if err == nil {
		t.Fatal("test_rets over ws: expected an error")
	}
	rpcErr, ok := err.(Error)
	if !ok {
		t.Fatalf("test_rets over ws: error %v is not an rpc.Error", err)
	}
	if rpcErr.ErrorCode() != -32601 {
		t.Errorf("test_rets over ws: code = %d, want -32601", rpcErr.ErrorCode())
	}
	// A bare namespace remains fully reachable over ws too.
	var n int
	if err := client.Call(&n, "nftest_echo", 7); err != nil {
		t.Errorf("nftest_echo over ws: unexpected error %v", err)
	}
}
