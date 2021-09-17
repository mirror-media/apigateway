package graph

import (
	"bytes"

	"github.com/olvrng/ujson"
)

func ReplaceNullString(input []byte) ([]byte, error) {
	b := make([]byte, 0, len(input))
	err := ujson.Walk(input, func(_ int, key, value []byte) bool {
		var isNullString bool
		if bytes.Equal(value, []byte("\"null\"")) {
			isNullString = true
		}
		// write to output
		if len(b) != 0 && ujson.ShouldAddComma(value, b[len(b)-1]) {
			b = append(b, ',')
		}
		if len(key) > 0 {
			b = append(b, key...)
			b = append(b, ':')
		}
		if isNullString {
			b = append(b, []byte("null")...)
		} else {
			b = append(b, value...)
		}
		return true
	})
	return b, err
}
