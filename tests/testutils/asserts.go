// Copyright 2024   Blue Wave Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils

import (
	"errors"
	"reflect"
	"testing"
)

func AssertEqual(t *testing.T, exp, act interface{}) {
	t.Helper()
	if !reflect.DeepEqual(exp, act) {
		t.Fatalf("\n\tExpect:\t%v\n\tGot:\t%v", exp, act)
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("No error expected, but got: %s", err)
	}
}

func AssertError(t *testing.T, err, expErr error) {
	t.Helper()
	if !errors.Is(err, expErr) {
		t.Fatalf("\n\tExpect error:\t%s\n\tGot:\t%s", expErr, err)
	}
}

func AssertNil(t *testing.T, act interface{}) {
	t.Helper()
	if !isNil(act) {
		t.Fatalf("\n\tExpect nil\n\tGot:\t%s", act)
	}
}

func isNil(object interface{}) bool {
	if object == nil {
		return true
	}

	value := reflect.ValueOf(object)
	kind := value.Kind()
	isNilableKind := containsKind(
		[]reflect.Kind{
			reflect.Chan, reflect.Func,
			reflect.Interface, reflect.Map,
			reflect.Ptr, reflect.Slice},
		kind)

	if isNilableKind && value.IsNil() {
		return true
	}

	return false
}

func containsKind(kinds []reflect.Kind, kind reflect.Kind) bool {
	for i := 0; i < len(kinds); i++ {
		if kind == kinds[i] {
			return true
		}
	}

	return false
}
