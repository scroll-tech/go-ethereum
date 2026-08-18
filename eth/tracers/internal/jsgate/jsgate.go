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

// Package jsgate holds the switch that turns the interpreted (JavaScript)
// tracer engines off.
//
// It sits in its own internal package so that the switch exported from
// eth/tracers can be one-way: nothing outside eth/tracers/... can turn the
// engines back on once a node has disabled them. Tests inside eth/tracers/...
// restore the default through ResetForTest.
package jsgate

import "sync/atomic"

// disabled is written once during node startup and read on every tracer
// lookup, so it is atomic rather than a plain bool.
var disabled atomic.Bool

// Disable turns the interpreted tracer engines off. There is deliberately no
// exported way to turn them back on.
func Disable() {
	disabled.Store(true)
}

// Disabled reports whether the interpreted tracer engines are off.
func Disabled() bool {
	return disabled.Load()
}

// ResetForTest turns the interpreted tracer engines back on. Only tests inside
// eth/tracers/... can call it, because this package is internal to eth/tracers.
func ResetForTest() {
	disabled.Store(false)
}
