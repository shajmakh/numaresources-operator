/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package features

import (
	"reflect"
	"testing"
)

func TestClone(t *testing.T) {
	type testCase struct {
		name   string
		topics Topics
	}

	testCases := []testCase{
		{
			name:   "empty topics",
			topics: Topics{},
		},
		{
			name:   "empty active topics",
			topics: Topics{Active: []string{}},
		},
		{
			name: "non empty active topics",
			topics: Topics{
				Active: []string{"1", "2", "3"},
			},
		},
	}

	for _, tc := range testCases {
		out := tc.topics.Clone()
		if !reflect.DeepEqual(out, tc.topics) {
			t.Errorf("%q failed to clone Topics, expected %+v got %+v", tc.name, tc.topics, out)
		}
	}
}
