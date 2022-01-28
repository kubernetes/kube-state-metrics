/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package namespacelist

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/labels"
)

// Parser parses the syntax for namespace lists
type Parser struct {
	source string
	pos    int
}

type UnexpectedSyntaxError struct {
	char string
	pos  int
}

// UnexpectedSyntaxError is returned when an invalid syntax is being parsed
func (err UnexpectedSyntaxError) Error() string {
	return fmt.Sprintf("syntax error, unexpected %v at col %v", err.char, err.pos)
}

func (parser *Parser) Parse() (map[string]labels.Selector, error) {
	reader := bufio.NewReader(strings.NewReader(parser.source))
	selectors := make(map[string]labels.Selector)

	namespace := strings.Builder{}

	err := readRunes(reader, func(r rune, eof bool, close func()) error {
		parser.pos++

		if eof {
			if namespace.Len() > 0 {
				selectors[namespace.String()] = labels.Everything()
			}
			close()
			return nil
		}

		if unicode.IsSpace(r) {
			// ignore whitespace
			return nil
		}

		if r == '=' {
			// foo-bar=[...], ...
			r, _, err := reader.ReadRune()
			if errors.Is(err, io.EOF) || r != '[' {
				// foo-bar=...
				return UnexpectedSyntaxError{char: "EOF", pos: parser.pos}
			} else if err != nil {
				return err
			}

			selector, err := parseSelector(parser, reader)
			if err != nil {
				return err
			}

			selectors[namespace.String()] = selector
			namespace.Reset()
			return nil
		}

		if r == ',' {
			// foo-bar, ...
			selectors[namespace.String()] = labels.Everything()
			namespace.Reset()
			return nil
		}

		namespace.WriteRune(r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return selectors, nil
}

func parseSelector(parser *Parser, reader *bufio.Reader) (labels.Selector, error) {
	str := strings.Builder{}

	err := readRunes(reader, func(r rune, eof bool, close func()) error {
		parser.pos++

		if eof {
			// a namespace selector should always be closed with a "]"
			return UnexpectedSyntaxError{char: "EOF", pos: parser.pos}
		}

		if r == ']' {
			// indicates the end of a namespace selector
			close()
			return nil
		}

		str.WriteRune(r)
		return nil
	})
	if err != nil {
		return nil, err
	}

	selector, err := labels.Parse(str.String())
	if err != nil {
		return nil, err
	}
	return selector, nil
}

func readRunes(r *bufio.Reader, reader func(r rune, eof bool, close func()) error) error {
	reading := true

	closer := func() {
		reading = false
	}

	for reading {
		r, _, err := r.ReadRune()

		if errors.Is(err, io.EOF) {
			err := reader(0, true, closer)
			if err != nil {
				return err
			}
			reading = false
			continue
		}
		if err != nil {
			return err
		}

		err = reader(r, false, closer)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewParser(source string) *Parser {
	return &Parser{source, 0}
}
