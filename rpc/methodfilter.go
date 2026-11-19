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
	"fmt"
	"sort"
	"strings"
)

// apiEntryMethodSeparator divides a namespace from a method name in an API list
// entry, as in "debug:executionWitness".
const apiEntryMethodSeparator = ":"

// MethodFilter decides which fully-qualified method names a Server may dispatch.
//
// An entry names either a bare namespace ("debug"), leaving every method in it
// reachable, or a single method ("debug:executionWitness"), which allows only
// the methods named for that namespace. The filter denies by default: a method
// whose namespace was never named is refused, so methods gained in a later
// rebase stay closed until somebody lists them.
//
// A Server with no filter allows everything, which is the default and keeps the
// behaviour of a plain namespace list unchanged.
type MethodFilter struct {
	// unrestricted holds namespaces every method of which is allowed.
	unrestricted map[string]struct{}
	// methods maps a namespace to the method names allowed within it.
	methods map[string]map[string]struct{}
}

// AllowMethod permits a single method. It is used for methods a Server
// registers on its own account, which therefore never appear in a
// caller-supplied API list.
func (f *MethodFilter) AllowMethod(namespace, method string) {
	if f == nil {
		return
	}
	if f.methods == nil {
		f.methods = make(map[string]map[string]struct{})
	}
	if f.methods[namespace] == nil {
		f.methods[namespace] = make(map[string]struct{})
	}
	f.methods[namespace][method] = struct{}{}
}

// isSubscriptionMethodName reports whether a fully-qualified method name would
// be routed to the subscription machinery rather than to a plain callback. It
// mirrors jsonrpcMessage.isSubscribe and isUnsubscribe.
func isSubscriptionMethodName(method string) bool {
	return strings.HasSuffix(method, subscribeMethodSuffix) ||
		strings.HasSuffix(method, unsubscribeMethodSuffix)
}

// Allows reports whether the given fully-qualified method, e.g.
// "debug_executionWitness", may be dispatched. A nil filter allows everything.
func (f *MethodFilter) Allows(method string) bool {
	if f == nil {
		return true
	}
	namespace, name, ok := splitMethodName(method)
	if !ok {
		return false
	}
	if _, ok := f.unrestricted[namespace]; ok {
		return true
	}
	_, ok = f.methods[namespace][name]
	return ok
}

// methodNames returns every individually allowed method, sorted by namespace
// then name. Namespaces allowed in full are not included, having no method
// names to report.
func (f *MethodFilter) methodNames() []string {
	var out []string
	for namespace, allowed := range f.methods {
		for name := range allowed {
			out = append(out, namespace+serviceMethodSeparator+name)
		}
	}
	sort.Strings(out)
	return out
}

// splitMethodName divides "debug_executionWitness" into its namespace and
// method name. It reports false if the name is not of that form.
func splitMethodName(method string) (namespace, name string, ok bool) {
	elem := strings.SplitN(method, serviceMethodSeparator, 2)
	if len(elem) != 2 || elem[0] == "" || elem[1] == "" {
		return "", "", false
	}
	return elem[0], elem[1], true
}

// ParseAPIEntries splits the entries of an API list into the namespaces to
// register and a filter carrying any method-level restrictions. The filter is
// nil when no entry named a method, leaving dispatch unchanged.
//
// Malformed and ambiguous entries are rejected rather than ignored, so a bad
// configuration stops the caller instead of quietly granting more access than
// was intended.
func ParseAPIEntries(entries []string) (namespaces []string, filter *MethodFilter, err error) {
	var (
		unrestricted = make(map[string]struct{})
		methods      = make(map[string]map[string]struct{})
		seen         = make(map[string]struct{})
	)
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.Split(entry, apiEntryMethodSeparator)
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		if len(parts) > 2 {
			return nil, nil, fmt.Errorf("invalid API entry %q: expected \"namespace\" or \"namespace:method\"", entry)
		}
		namespace := parts[0]
		if namespace == "" {
			return nil, nil, fmt.Errorf("invalid API entry %q: empty namespace", entry)
		}
		if _, ok := seen[namespace]; !ok {
			seen[namespace] = struct{}{}
			namespaces = append(namespaces, namespace)
		}
		if len(parts) == 1 {
			unrestricted[namespace] = struct{}{}
			continue
		}
		method := parts[1]
		if method == "" {
			return nil, nil, fmt.Errorf("invalid API entry %q: empty method name", entry)
		}
		// Any name ending in the subscribe suffix is routed to handleSubscribe,
		// which takes the namespace from the first segment of the method name and
		// the subscription name from the request body. Neither is the name matched
		// here, so such an entry would open every subscription of the namespace
		// while looking specific. Test the whole method name, exactly as the
		// dispatcher does, not just the part after the colon.
		if isSubscriptionMethodName(namespace + serviceMethodSeparator + method) {
			return nil, nil, fmt.Errorf("invalid API entry %q: subscriptions cannot be allowed per method; list the %q namespace in full if you need them", entry, namespace)
		}
		if methods[namespace] == nil {
			methods[namespace] = make(map[string]struct{})
		}
		methods[namespace][method] = struct{}{}
	}
	// Listing a namespace both bare and per-method is contradictory. Rather than
	// guessing which was meant, refuse to start.
	for namespace := range methods {
		if _, ok := unrestricted[namespace]; ok {
			return nil, nil, fmt.Errorf("ambiguous API list: namespace %q is listed both bare and per-method; remove one", namespace)
		}
	}
	if len(methods) == 0 {
		return namespaces, nil, nil
	}
	return namespaces, &MethodFilter{unrestricted: unrestricted, methods: methods}, nil
}
