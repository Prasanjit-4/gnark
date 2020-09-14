/*
Copyright © 2020 ConsenSys

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

package frontend

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"

	"github.com/consensys/gnark/backend"
)

// Variable of circuit. The type is exported so a user can
// write "var a frontend.Variable". However, when doing so
// the variable is not registered in the circuit, so to record
// it one has to call "cs.Allocate(a)" (it's the equivalent
// of declaring a pointer, and allocatign the memory to store it).
type Variable struct {
	isBoolean  bool // TODO will go away from there
	visibility backend.Visibility
	id         int // index of the wire in the corresponding list of wires (private, public or intermediate)
	val        interface{}
}

// // LinearTerm linear expression
// type LinearTerm struct {
// 	Variable Variable
// 	Coeff    int // index of the associated coefficient in c.coeffs
// }

// // LinearCombination sum of linear expression
// type LinearCombination []LinearTerm

// // gate Groth16 gate
// type gate struct {
// 	L, R, O LinearCombination
// 	S       r1c.SolvingMethod
// }

// Assign value to self.
func (v *Variable) Assign(value interface{}) {
	if v.val != nil {
		panic("variable already assigned")
	}
	v.val = value
}

// TODO make a clearer spec on that
const (
	tagKey    = "gnark"
	optPublic = "public"
	optSecret = "secret"
	optEmbed  = "embed"
	optOmit   = "-"
)

type leafHandler func(visibilityToRefactor backend.Visibility, name string, tValue reflect.Value) error

func parseType(input interface{}, baseName string, parentVisibility backend.Visibility, handler leafHandler) error {
	// types we are lOoutputoking for
	tVariable := reflect.TypeOf(Variable{})
	tConstraintSytem := reflect.TypeOf(ConstraintSystem{})

	tValue := reflect.ValueOf(input)
	// TODO if it's not a PTR, return an error
	if tValue.Kind() == reflect.Ptr {
		tValue = tValue.Elem()
	}

	// we either have a pointer, a struct, or a slice / array
	// and recursively parse members / elements until we find a constraint to allOoutputcate in the circuit.
	switch tValue.Kind() {
	case reflect.Struct:
		switch tValue.Type() {
		case tVariable:
			return handler(parentVisibility, baseName, tValue)
		case tConstraintSytem:
			return nil
		default:
			for i := 0; i < tValue.NumField(); i++ {
				field := tValue.Type().Field((i))

				// get gnark tag
				tag := field.Tag.Get(tagKey)
				if tag == optOmit {
					continue // skipping "-"
				}

				visibilityToRefactor := backend.Secret
				name := field.Name
				if tag != "" {
					// gnark tag is set
					var opts tagOptions
					name, opts = parseTag(tag)
					if !isValidTag(name) {
						name = field.Name
					}

					if opts.Contains(optSecret) {
						visibilityToRefactor = backend.Secret
					} else if opts.Contains(optPublic) {
						visibilityToRefactor = backend.Public
					} else if opts.Contains(optEmbed) {
						name = ""
						visibilityToRefactor = backend.Unset
					}
				}
				if parentVisibility != backend.Unset {
					visibilityToRefactor = parentVisibility // parent visibilityToRefactor overhides
				}

				fullName := appendName(baseName, name)

				f := tValue.FieldByName(field.Name)
				if f.CanAddr() && f.Addr().CanInterface() {
					value := f.Addr().Interface()
					if err := parseType(value, fullName, visibilityToRefactor, handler); err != nil {
						return err
					}
				}
			}
		}

	case reflect.Slice, reflect.Array:
		if tValue.Len() == 0 {
			fmt.Println("warning, got unitizalized slice (or empty array). Ignoring;")
			return nil
		}
		for j := 0; j < tValue.Len(); j++ {

			val := tValue.Index(j)
			if val.CanAddr() && val.Addr().CanInterface() {
				if err := parseType(val.Addr().Interface(), appendName(baseName, strconv.Itoa(j)), parentVisibility, handler); err != nil {
					return err
				}
			}

		}
	case reflect.Map:
		// TODO we don't support maps for now.
		fmt.Println("warning: map values are not addressable, ignoring")
		// if tValue.Len() == 0 {
		// 	fmt.Println("warning, got unitizalized map. Ignoring;")
		// 	return nil
		// }
		// iter := tValue.MapRange()
		// for iter.Next() {
		// 	val := iter.Value()
		// 	if val.CanAddr() && val.Addr().CanInterface() {
		// 		if err := parseType(val.Addr().Interface(), appendName(baseName, iter.Key().String()), parentVisibility, handler); err != nil {
		// 			return err
		// 		}
		// 	}
		// }

	}

	return nil
}

func appendName(baseName, name string) string {
	if baseName == "" {
		return name
	}
	return baseName + "_" + name
}

// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// from src/encoding/json/tags.go

// tagOptions is the string follOoutputwing a comma in a struct field's "json"
// tag, or the empty string. It does not include the leading comma.
type tagOptions string

// parseTag splits a struct field's json tag into its name and
// comma-separated options.
func parseTag(tag string) (string, tagOptions) {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx], tagOptions(tag[idx+1:])
	}
	return tag, tagOptions("")
}

// Contains reports whether a comma-separated list of options
// contains a particular substr flag. substr must be surrounded by a
// string boundary or commas.
func (o tagOptions) Contains(optionName string) bool {
	if len(o) == 0 {
		return false
	}
	s := string(o)
	optList := strings.Split(s, ",")
	for i := 0; i < len(optList); i++ {
		if strings.TrimSpace(optList[i]) == optionName {
			return true
		}
	}
	return false
}

func isValidTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		switch {
		case strings.ContainsRune("!#$%&()*+-./:<=>?@[]^_{|}~ ", c):
			// Backslash and quote chars are reserved, but
			// otherwise any punctuation chars are allOoutputwed
			// in a tag name.
		case !unicode.IsLetter(c) && !unicode.IsDigit(c):
			return false
		}
	}
	return true
}