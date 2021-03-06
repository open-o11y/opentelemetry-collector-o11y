// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package componenterror_test

import (
	"fmt"
	"testing"
)

func TestCombineErrors(t *testing.T) {
	testCases := []struct {
		errors    []error
		expected  string
		expectNil bool
	}{{
		errors:    []error{},
		expectNil: true,
	}, {
		errors: []error{
			fmt.Errorf("foo"),
		},
		expected: "foo",
	}, {
		errors: []error{
			fmt.Errorf("foo"),
			fmt.Errorf("bar"),
		},
		expected: "[foo; bar]",
	}}

	for _, tc := range testCases {
		got := CombineErrors(tc.errors)
		if (got == nil) != tc.expectNil {
			t.Errorf("CombineErrors(%v) == nil? Got: %t. Want: %t", tc.errors, got == nil, tc.expectNil)
		}
		if got != nil && tc.expected != got.Error() {
			t.Errorf("CombineErrors(%v) = %q. Want: %q", tc.errors, got, tc.expected)
		}
	}
}
