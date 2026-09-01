// Copyright 2019 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

// Package utils contains internal helper functions for go-ethereum commands.
package utils

import (
	"reflect"
	"testing"
)

func Test_SplitTagsFlag(t *testing.T) {
	tests := []struct {
		name string
		args string
		want map[string]string
	}{
		{
			"2 tags case",
			"host=localhost,bzzkey=123",
			map[string]string{
				"host":   "localhost",
				"bzzkey": "123",
			},
		},
		{
			"1 tag case",
			"host=localhost123",
			map[string]string{
				"host": "localhost123",
			},
		},
		{
			"empty case",
			"",
			map[string]string{},
		},
		{
			"garbage",
			"smth=smthelse=123",
			map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SplitTagsFlag(tt.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTagsFlag() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestJSTracerFlagsConflict pins the precedence between the deprecated
// --rpc.disable-js-tracers and its replacement. The deprecated flag can no
// longer enable anything, so it only conflicts with an explicit allow.
func TestJSTracerFlagsConflict(t *testing.T) {
	for _, tt := range []struct {
		name                string
		disableSet, disable bool
		allowSet, allow     bool
		conflict            bool
	}{
		{name: "neither flag given"},
		{name: "deprecated disable alone", disableSet: true, disable: true},
		{name: "explicit allow alone", allowSet: true, allow: true},
		{name: "explicit deny alone", allowSet: true},
		{name: "disable plus deny agree", disableSet: true, disable: true, allowSet: true},
		{name: "disable=false plus allow agree", disableSet: true, allowSet: true, allow: true},
		{name: "disable plus allow contradict", disableSet: true, disable: true, allowSet: true, allow: true, conflict: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := jsTracerFlagsConflict(tt.disableSet, tt.disable, tt.allowSet, tt.allow); got != tt.conflict {
				t.Errorf("jsTracerFlagsConflict() = %v, want %v", got, tt.conflict)
			}
		})
	}
}
