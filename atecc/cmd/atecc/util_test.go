package main

import (
	"bytes"
	"testing"
)

func TestPrettyHexIndent(t *testing.T) {
	testCases := []struct {
		name   string
		in     []byte
		prefix string
		space  string
		want   string
	}{
		{"empty", []byte{}, "  ", "", ""},
		{"one", []byte{0x00}, "  ", "", "  00"},
		{"two", []byte{0x00, 0x01}, "  ", "", "  00 01"},
		{"three", []byte{0x00, 0x01, 0x02}, "    ", "", "    00 01 02"},
		{
			"big", bytes.Repeat([]byte{0x00}, 32), "    ", "",
			"    00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00\n" +
				"    00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00",
		},
		{
			"space", bytes.Repeat([]byte{0x00}, 32), "    ", " ",
			"    00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00\n" +
				"    00 00 00 00 00 00 00 00  00 00 00 00 00 00 00 00",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := prettyHexIndent(tc.in, tc.prefix, tc.space)
			if got != tc.want {
				t.Errorf("want %q, got %q", tc.want, got)
			}
		})
	}
}
