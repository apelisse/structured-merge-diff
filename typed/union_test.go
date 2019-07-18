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

package typed_test

import (
	"testing"

	"sigs.k8s.io/structured-merge-diff/typed"
)

var unionParser = func() typed.ParseableType {
	parser, err := typed.NewParser(`types:
- name: union
  map:
    fields:
    - name: discriminator
      type:
        scalar: string
    - name: one
      type:
        scalar: numeric
    - name: two
      type:
        scalar: numeric
    - name: three
      type:
        scalar: numeric
    - name: letter
      type:
        scalar: string
    - name: a
      type:
        scalar: numeric
    - name: b
      type:
        scalar: numeric
    unions:
    - discriminator: discriminator
      deduceDiscriminator: true
      fields:
      - fieldName: one
        discriminatorValue: One
      - fieldName: two
        discriminatorValue: TWO
      - fieldName: three
        discriminatorValue: three
    - discriminator: letter
      fields:
      - fieldName: a
        discriminatorValue: A
      - fieldName: b
        discriminatorValue: b`)
	if err != nil {
		panic(err)
	}
	return parser.Type("union")
}()

func TestNormalizeUnions(t *testing.T) {
	tests := []struct {
		name string
		old  typed.YAMLObject
		new  typed.YAMLObject
		out  typed.YAMLObject
	}{
		{
			name: "nothing changed, add discriminator",
			new:  `{"one": 1}`,
			out:  `{"one": 1, "discriminator": "One"}`,
		},
		{
			name: "nothing changed, non-deduced",
			new:  `{"a": 1}`,
			out:  `{"a": 1}`,
		},
		{
			name: "proper union update, setting discriminator",
			new:  `{"two": 1}`,
			out:  `{"two": 1, "discriminator": "TWO"}`,
		},
		{
			name: "proper union update, non-deduced",
			new:  `{"b": 1}`,
			out:  `{"b": 1}`,
		},
		{
			name: "proper union update from not-set, setting discriminator",
			new:  `{"two": 1}`,
			out:  `{"two": 1, "discriminator": "TWO"}`,
		},
		{
			name: "proper union update, non-deduced",
			new:  `{"b": 1}`,
			out:  `{"b": 1}`,
		},
		{
			name: "remove union, with discriminator",
			new:  `{}`,
			out:  `{}`,
		},
		{
			name: "remove union, not discriminator, non-deduced",
			new:  `{"letter": "A"}`,
			out:  `{"letter": "A"}`,
		},
		{
			name: "change discriminator, nothing else",
			new:  `{"discriminator": "random"}`,
			out:  `{"discriminator": "random"}`,
		},
		{
			name: "change discriminator, nothing else, non-deduced",
			new:  `{"letter": "b"}`,
			out:  `{"letter": "b"}`,
		},
		{
			name: "set discriminator and other field, clean other field",
			new:  `{"letter": "b", "a": 1}`,
			out:  `{"letter": "b"}`,
		},
		{
			name: "Non-deduced discriminator is not deduced",
			new:  `{"b": 1}`,
			out:  `{"b": 1}`,
		},
		{
			name: "Nothing set, nothing deduced",
			new:  `{}`,
			out:  `{}`,
		},
		{
			name: "deduced discriminator is set",
			new:  `{"one": 1}`,
			out:  `{"one": 1, "discriminator": "One"}`,
		},
		{
			name: "deduce discriminator doesn't match, re-deduced",
			new:  `{"one": 1, "discriminator": "Two"}`,
			out:  `{"one": 1, "discriminator": "One"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			new, err := unionParser.FromYAML(test.new)
			if err != nil {
				t.Fatalf("failed to parse new object: %v", err)
			}
			out, err := unionParser.FromYAML(test.out)
			if err != nil {
				t.Fatalf("failed to parse out object: %v", err)
			}
			got, err := new.NormalizeUnions()
			if err != nil {
				t.Fatalf("failed to normalize unions: %v", err)
			}
			comparison, err := out.Compare(got)
			if err != nil {
				t.Fatalf("failed to compare result and expected: %v", err)
			}
			if !comparison.IsSame() {
				t.Errorf("Result is different from expected:\n%v", comparison)
			}
		})
	}
}

func TestNormalizeUnionError(t *testing.T) {
	tests := []struct {
		name string
		new  typed.YAMLObject
	}{
		{
			name: "Multiple fields set, no discriminator",
			new:  `{"one": 2, "two": 1}`,
		},
		{
			name: "Multiple fields set and deduce-discriminator",
			new:  `{"discriminator": "One", "one": 1, "two": 1, "three": 1}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			new, err := unionParser.FromYAML(test.new)
			if err != nil {
				t.Fatalf("failed to parse new object: %v", err)
			}
			_, err = new.NormalizeUnions()
			if err == nil {
				t.Fatal("Normalization should have failed, but hasn't.")
			}
		})
	}
}
