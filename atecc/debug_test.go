package atecc

import (
	"fmt"
	"testing"
)

func TestHexDump(t *testing.T) {
	want := "h -> \n00000000  66 6f 6f 62 61 72                                 |foobar|\n\n <- h"
	got := fmt.Sprintf("h -> %s <- h", hexDump([]byte("foobar")))
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}
