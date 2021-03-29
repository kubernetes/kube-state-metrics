/*
Copyright 2016 Google Inc. All rights reserved.

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

package parser

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"unicode/utf8"

	"github.com/google/go-jsonnet/ast"
	"github.com/google/go-jsonnet/internal/errors"
)

// StringUnescape compiles out the escape codes in the string
func StringUnescape(loc *ast.LocationRange, s string) (string, error) {
	var buf bytes.Buffer
	// read one rune at a time
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		i += w
		switch r {
		case '\\':
			if i >= len(s) {
				return "", errors.MakeStaticError("Truncated escape sequence in string literal.", *loc)
			}
			r2, w := utf8.DecodeRuneInString(s[i:])
			i += w
			switch r2 {
			case '"':
				buf.WriteRune('"')
			case '\'':
				buf.WriteRune('\'')
			case '\\':
				buf.WriteRune('\\')
			case '/':
				buf.WriteRune('/') // See json.org, \/ is a valid escape.
			case 'b':
				buf.WriteRune('\b')
			case 'f':
				buf.WriteRune('\f')
			case 'n':
				buf.WriteRune('\n')
			case 'r':
				buf.WriteRune('\r')
			case 't':
				buf.WriteRune('\t')
			case 'u':
				if i+4 > len(s) {
					return "", errors.MakeStaticError("Truncated unicode escape sequence in string literal.", *loc)
				}
				codeBytes, err := hex.DecodeString(s[i : i+4])
				if err != nil {
					return "", errors.MakeStaticError(fmt.Sprintf("Unicode escape sequence was malformed: %s", s[0:4]), *loc)
				}
				code := int(codeBytes[0])*256 + int(codeBytes[1])
				buf.WriteRune(rune(code))
				i += 4
			default:
				return "", errors.MakeStaticError(fmt.Sprintf("Unknown escape sequence in string literal: \\%c", r2), *loc)
			}

		default:
			buf.WriteRune(r)
		}
	}
	return buf.String(), nil
}

// StringEscape does the opposite of StringUnescape
func StringEscape(s string, single bool) string {
	var buf bytes.Buffer
	// read one rune at a time
	for i := 0; i < len(s); {
		r, w := utf8.DecodeRuneInString(s[i:])
		i += w
		switch r {
		case '"':
			if !single {
				buf.WriteRune('\\')
			}
			buf.WriteRune(r)
		case '\'':
			if single {
				buf.WriteRune('\\')
			}
			buf.WriteRune(r)
		case '\\':
			buf.WriteRune('\\')
			buf.WriteRune(r)
		case '\b':
			buf.WriteRune('\\')
			buf.WriteRune('b')
		case '\f':
			buf.WriteRune('\\')
			buf.WriteRune('f')
		case '\n':
			buf.WriteRune('\\')
			buf.WriteRune('n')
		case '\r':
			buf.WriteRune('\\')
			buf.WriteRune('r')
		case '\t':
			buf.WriteRune('\\')
			buf.WriteRune('t')
		case '\u0000':
			buf.WriteString("\\u0000")

		default:
			if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
				buf.WriteRune('\\')
				buf.WriteRune('u')
				buf.Write([]byte(fmt.Sprintf("%04x", int(r))))
			} else {
				buf.WriteRune(r)
			}
		}
	}
	return buf.String()
}
