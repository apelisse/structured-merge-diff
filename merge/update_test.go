/*
Copyright 2018 The Kubernetes Authors.

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

package merge_test

import (
	"fmt"
	"testing"

	"sigs.k8s.io/structured-merge-diff/merge"
	"sigs.k8s.io/structured-merge-diff/typed"
)

// State of the current test in terms of live object. One can check at
// any time that Live and Owners match the expectations.
type State struct {
	Live   *typed.TypedValue
	Parser *typed.Parser
	// Typename is the typename used to create objects in the
	// schema.
	Typename string
	Owners   merge.Owners
	Updater  *merge.Updater
}

func (s *State) checkInit() error {
	if s.Owners == nil {
		s.Owners = merge.Owners{}
	}
	if s.Live == nil {
		obj, err := s.Parser.NewEmpty(s.Typename)
		if err != nil {
			return fmt.Errorf("failed to create new empty object: %v", err)
		}
		s.Live = &obj
		fmt.Println("Created live:", s.Live)
	}
	return nil
}

// Update the current state with the passed in object
func (s *State) Update(obj typed.YAMLObject, owner string) error {
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.Parser.FromYAML(obj, s.Typename)
	owners, err := s.Updater.Update(*s.Live, tv, s.Owners, owner)
	if err != nil {
		return err
	}
	s.Live = &tv
	s.Owners = owners

	return nil
}

// Apply the passed in object to the current state
func (s *State) Apply(obj typed.YAMLObject, owner string, force bool) error {
	if err := s.checkInit(); err != nil {
		return err
	}
	tv, err := s.Parser.FromYAML(obj, s.Typename)
	if err != nil {
		return err
	}
	new, owners, err := s.Updater.Apply(*s.Live, tv, s.Owners, owner, force)
	if err != nil {
		return err
	}
	s.Live = &new
	s.Owners = owners

	return nil
}

func (s *State) CompareLive(obj typed.YAMLObject) (*typed.Comparison, error) {
	if err := s.checkInit(); err != nil {
		return nil, err
	}
	tv, err := s.Parser.FromYAML(obj, s.Typename)
	if err != nil {
		return nil, err
	}
	return s.Live.Compare(tv)
}

// TestExample shows how to use the test framework
func TestExample(t *testing.T) {
	parser, err := typed.NewParser(`types:
- name: lists
  struct:
    fields:
    - name: list
      type:
        list:
          elementType:
            scalar: string
          elementRelationship: associative`)
	if err != nil {
		t.Fatalf("Failed to create parser: %v", err)
	}
	state := &State{
		Updater:  &merge.Updater{},
		Parser:   parser,
		Typename: "lists",
	}

	config := typed.YAMLObject(`
list:
- a
- b
- c
`)
	err = state.Apply(config, "default", false)
	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	config = typed.YAMLObject(`
list:
- a
- b
- c
- d`)
	err = state.Apply(config, "default", false)

	if err != nil {
		t.Fatalf("Wanted err = %v, got %v", nil, err)
	}

	// The following is wrong because the code doesn't work yet.
	_, err = state.CompareLive(config)
	if err == nil {
		t.Fatalf("Succeeded to compare live with config")
	}
}
