// Copyright 2017 The go-ethereum Authors
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

// Package tracers is a manager for transaction tracing engines.
package tracers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/vm"
	"github.com/scroll-tech/go-ethereum/eth/tracers/internal/jsgate"
)

// Context contains some contextual infos for a transaction execution that is not
// available from within the EVM object.
type Context struct {
	BlockNumber uint64      // Block number of the block the tx is contained within (zero if dangling tx or call)
	BlockHash   common.Hash // Hash of the block the tx is contained within (zero if dangling tx or call)
	TxIndex     int         // Index of the transaction within a block (zero if dangling tx or call)
	TxHash      common.Hash // Hash of the transaction being traced (zero if dangling call)
}

// Tracer interface extends vm.EVMLogger and additionally
// allows collecting the tracing result.
type Tracer interface {
	vm.EVMLogger
	GetResult() (json.RawMessage, error)
	// Stop terminates execution of the tracer at the first opportune moment.
	Stop(err error)
}

type lookupFunc func(string, *Context) (Tracer, error)

var (
	// lookups serves the named tracers and is consulted first. wildcardLookups
	// holds the interpreted engines and is consulted last, because those evaluate
	// dynamic user-supplied code.
	lookups         []lookupFunc
	wildcardLookups []lookupFunc
)

var (
	errTracerNotFound = errors.New("tracer not found")

	// ErrJSTracersDisabled is returned when the requested tracer is not a native
	// one and the interpreted engines that could have served it are turned off.
	// The name may also simply be unknown, which is why it wraps errTracerNotFound.
	ErrJSTracersDisabled = fmt.Errorf("%w: JavaScript tracers are disabled, only native tracers are available", errTracerNotFound)
)

// DisableJSTracers turns the interpreted tracer engines off. The native Go
// tracers and the default struct logger keep working. Call it during startup,
// before the RPC layer begins serving requests.
//
// There is deliberately no exported counterpart to turn them back on, so no
// later call can undo it. The setting is process-wide, matching the registry
// it gates.
func DisableJSTracers() {
	jsgate.Disable()
}

// JSTracersDisabled reports whether the interpreted tracer engines are off.
func JSTracersDisabled() bool {
	return jsgate.Disabled()
}

// RegisterLookup registers a method as a lookup for tracers, meaning that
// users can invoke a named tracer through that lookup. If 'wildcard' is true,
// then the lookup will be placed last, and it will be skipped entirely when
// JavaScript tracers are disabled. This is meant for interpreted engines (js)
// which can evaluate dynamic user-supplied code.
func RegisterLookup(wildcard bool, lookup lookupFunc) {
	if wildcard {
		wildcardLookups = append(wildcardLookups, lookup)
	} else {
		lookups = append([]lookupFunc{lookup}, lookups...)
	}
}

// New returns a new instance of a tracer, by iterating through the registered
// lookups. The interpreted engines are skipped when they are disabled, so
// user-supplied code is never evaluated.
func New(code string, ctx *Context) (Tracer, error) {
	if tracer := tryLookups(lookups, code, ctx); tracer != nil {
		return tracer, nil
	}
	if jsgate.Disabled() {
		return nil, ErrJSTracersDisabled
	}
	if tracer := tryLookups(wildcardLookups, code, ctx); tracer != nil {
		return tracer, nil
	}
	return nil, errTracerNotFound
}

// tryLookups returns the first tracer the given lookups can serve, or nil if
// none of them recognise the code.
func tryLookups(lookups []lookupFunc, code string, ctx *Context) Tracer {
	for _, lookup := range lookups {
		if tracer, err := lookup(code, ctx); err == nil {
			return tracer
		}
	}
	return nil
}

const (
	memoryPadLimit = 1024 * 1024
)

// GetMemoryCopyPadded returns offset + size as a new slice.
// It zero-pads the slice if it extends beyond memory bounds.
func GetMemoryCopyPadded(m *vm.Memory, offset, size int64) ([]byte, error) {
	if offset < 0 || size < 0 {
		return nil, errors.New("offset or size must not be negative")
	}
	if int(offset+size) < m.Len() { // slice fully inside memory
		return m.GetCopy(offset, size), nil
	}
	paddingNeeded := int(offset+size) - m.Len()
	if paddingNeeded > memoryPadLimit {
		return nil, fmt.Errorf("reached limit for padding memory slice: %d", paddingNeeded)
	}
	cpy := make([]byte, size)
	if overlap := int64(m.Len()) - offset; overlap > 0 {
		copy(cpy, m.GetPtr(offset, overlap))
	}
	return cpy, nil
}
