/*
Copyright 2019 The Kubernetes Authors.

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

package value

import (
	"testing"
)

func TestValueLess(t *testing.T) {
	table := []struct {
		name string
		// we expect a < b and !(b < a) unless eq is true, in which
		// case we expect less to return false in both orders.
		a, b interface{}
		eq   bool
	}{
		{
			name: "Float-1",
			a:    1.14,
			b:    3.14,
		}, {
			name: "Float-2",
			a:    1.,
			b:    1.,
			eq:   true,
		}, {
			name: "Float-3",
			a:    1.,
			b:    1,
			eq:   true,
		}, {
			name: "Float-4",
			a:    1.,
			b:    2,
		}, {
			name: "Float-5",
			a:    1.,
			b:    "aoeu",
		}, {
			name: "Float-6",
			a:    1.,
			b:    true,
		}, {
			name: "Float-7",
			a:    1.,
			b:    []interface{}{},
		}, {
			name: "Float-8",
			a:    1.,
			b:    map[string]interface{}{},
		}, {
			name: "Float-9",
			a:    1.,
			b:    nil,
		}, {
			name: "Int-1",
			a:    1,
			b:    2,
		}, {
			name: "Int-2",
			a:    1,
			b:    1,
			eq:   true,
		}, {
			name: "Int-3",
			a:    1,
			b:    1.,
			eq:   true,
		}, {
			name: "Int-4",
			a:    1,
			b:    2.,
		}, {
			name: "Int-5",
			a:    1,
			b:    "aoeu",
		}, {
			name: "Int-6",
			a:    1,
			b:    true,
		}, {
			name: "Int-7",
			a:    1,
			b:    []interface{}{},
		}, {
			name: "Int-8",
			a:    1,
			b:    map[string]interface{}{},
		}, {
			name: "Int-9",
			a:    1,
			b:    nil,
		}, {
			name: "String-1",
			a:    "b-12",
			b:    "b-9",
		}, {
			name: "String-2",
			a:    "folate",
			b:    "folate",
			eq:   true,
		}, {
			name: "String-3",
			a:    "folate",
			b:    true,
		}, {
			name: "String-4",
			a:    "folate",
			b:    []interface{}{},
		}, {
			name: "String-5",
			a:    "folate",
			b:    map[string]interface{}{},
		}, {
			name: "String-6",
			a:    "folate",
			b:    nil,
		}, {
			name: "Bool-1",
			a:    false,
			b:    true,
		}, {
			name: "Bool-2",
			a:    false,
			b:    false,
			eq:   true,
		}, {
			name: "Bool-3",
			a:    true,
			b:    true,
			eq:   true,
		}, {
			name: "Bool-4",
			a:    false,
			b:    []interface{}{},
		}, {
			name: "Bool-5",
			a:    false,
			b:    map[string]interface{}{},
		}, {
			name: "Bool-6",
			a:    false,
			b:    nil,
		}, {
			name: "List-1",
			a:    []interface{}{},
			b:    []interface{}{},
			eq:   true,
		}, {
			name: "List-2",
			a:    []interface{}{1},
			b:    []interface{}{1},
			eq:   true,
		}, {
			name: "List-3",
			a:    []interface{}{1},
			b:    []interface{}{2},
		}, {
			name: "List-4",
			a:    []interface{}{1},
			b:    []interface{}{1, 1},
		}, {
			name: "List-5",
			a:    []interface{}{1, 1},
			b:    []interface{}{2},
		}, {
			name: "List-6",
			a:    []interface{}{},
			b:    map[string]interface{}{},
		}, {
			name: "List-7",
			a:    []interface{}{},
			b:    nil,
		}, {
			name: "Map-1",
			a:    map[string]interface{}{"carotine": 1},
			b:    map[interface{}]interface{}{"carotine": 1},
			eq:   true,
		}, {
			name: "Map-2",
			a:    map[string]interface{}{"carotine": 1},
			b:    map[string]interface{}{"carotine": 2},
		}, {
			name: "Map-3",
			a:    map[string]interface{}{"carotine": 1},
			b:    map[string]interface{}{"ethanol": 1},
		}, {
			name: "Map-4",
			a:    map[string]interface{}{"carotine": 1},
			b:    map[string]interface{}{"ethanol": 1, "carotine": 2},
		}, {
			name: "Map-5",
			a:    map[string]interface{}{"carotine": 1},
			b:    map[string]interface{}{"carotine": 1, "ethanol": 1},
		}, {
			name: "Map-6",
			a:    map[string]interface{}{"carotine": 1, "ethanol": 1},
			b:    map[string]interface{}{"carotine": 2},
		}, {
			name: "Map-7",
			a:    map[string]interface{}{},
			b:    nil,
		},
	}

	for i := range table {
		i := i
		t.Run(table[i].name, func(t *testing.T) {
			tt := table[i]
			a, b := ValueInterface{Value: tt.a}, ValueInterface{Value: tt.b}
			if tt.eq {
				if Less(a, b) {
					t.Errorf("oops, a < b: %#v, %#v", a, b)
				}
			} else {
				if !Less(a, b) {
					t.Errorf("oops, a >= b: %#v, %#v", a, b)
				}
			}
			if Less(b, a) {
				t.Errorf("oops, b < a: %#v, %#v", b, a)
			}
		})
	}

}
