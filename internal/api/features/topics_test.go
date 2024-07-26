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
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestGetTopics(t *testing.T) {
	var expected, actual TopicInfo

	topicsData, err := os.ReadFile("_topics.json")
	if err != nil {
		t.Error(err)
	}

	_ = json.Unmarshal(topicsData, &expected)
	actual = GetTopics()

	if !reflect.DeepEqual(actual.Active, expected.Active) {
		t.Errorf("embedded supported topics are not as expected, expected:\n%+v\nfound:\n%+v", expected, actual)
	}
}

func TestBuildFilterConsistAny(t *testing.T) {
	type testcase struct {
		name     string
		key      string
		values   []string
		expected string
	}
	testCases := []testcase{
		{
			name:     "two values",
			key:      "app",
			values:   []string{"foo", "bar"},
			expected: "app: consistAny {foo,bar}",
		},
		{
			name:     "one value",
			key:      "app",
			values:   []string{"foo"},
			expected: "app: consistAny {foo}",
		},
		{
			name:     "empty key",
			key:      "  ",
			values:   []string{"foo", "bar"},
			expected: "",
		},
		{
			name:     "empty values",
			key:      "app",
			values:   []string{},
			expected: "app: consistAny {}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			got, _ := BuildFilterConsistAny(tc.key, tc.values)
			if got != tc.expected {
				t.Errorf("expected %q got %q", tc.expected, got)
			}
		})
	}
}
