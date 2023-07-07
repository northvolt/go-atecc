package atecc

import (
	"bytes"
	"encoding/hex"
	"strconv"
	"testing"
)

func TestPackets(t *testing.T) {
	testCases := []struct {
		p *packet
		b []byte
	}{
		{
			must(newInfoCommand(infoModeRevision)),
			[]byte{0x7, 0x30, 0x0, 0x0, 0x0, 0x03, 0x5d},
		},
	}

	for i, tc := range testCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			var enc packetEncoder
			b, err := enc.Encode(tc.p)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, tc.b) {
				t.Error(hex.Dump(b))
				t.Error(hex.Dump(tc.b))
			}
		})
	}
}

func must(p *packet, err error) *packet {
	if err != nil {
		panic(err)
	}
	return p
}
