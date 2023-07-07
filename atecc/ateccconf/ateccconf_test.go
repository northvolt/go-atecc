package ateccconf

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
)

var (
	// flagWriteTestdata is used to write test data based on test state.
	//
	// This only works when you test this specific package (not with path/...).
	flagWriteTestdata = flag.Bool("write-atecc-testdata", false, "write atecc testdata")
)

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(m.Run())
}

var golden608 = append(
	// 16 first bytes are static inside of the device
	[]byte{
		0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7,
		0x8, 0x9, 0xa, 0xb, 0xc, 0xd, 0xe, 0xf,
	}, Default608...,
)

func TestUnmarshal(t *testing.T) {
	var (
		want = Config608{
			SN03:       [4]byte{0x0, 0x1, 0x2, 0x3},
			RevNum:     [4]byte{0x4, 0x5, 0x6, 0x7},
			SN48:       [5]byte{0x8, 0x9, 0xa, 0xb, 0xc},
			AESEnable:  AESEnable{Bits: 0xd},
			I2CEnable:  I2CEnable{Bits: 0xe},
			Reserved15: 0xf,

			// Default608...
			I2CAddress: 0x6a,
			Reserved17: 0,
			CountMatch: CountMatch{Bits: 0},
			ChipMode:   ChipMode608{Bits: 1},
			SlotConfig: [16]SlotConfig{
				{Bits1: 0x85, Bits2: 0x00},
				{Bits1: 0x82, Bits2: 0x00},
				{Bits1: 0x85, Bits2: 0x20},
				{Bits1: 0x85, Bits2: 0x20},
				{Bits1: 0x85, Bits2: 0x20},
				{Bits1: 0xc6, Bits2: 0x46},
				{Bits1: 0x8f, Bits2: 0x0f},
				{Bits1: 0x9f, Bits2: 0x8f},
				{Bits1: 0x0f, Bits2: 0x0f},
				{Bits1: 0x8f, Bits2: 0x0f},
				{Bits1: 0x0f, Bits2: 0x0f},
				{Bits1: 0x0f, Bits2: 0x0f},
				{Bits1: 0x0f, Bits2: 0x0f},
				{Bits1: 0x0f, Bits2: 0x0f},
				{Bits1: 0x0d, Bits2: 0x1f},
				{Bits1: 0x0f, Bits2: 0x0f},
			},

			Counter: [2]Counter{
				{Value: [8]uint8{0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0x0, 0x0}},
				{Value: [8]uint8{0xff, 0xff, 0xff, 0xff, 0x0, 0x0, 0x0, 0x0}},
			},

			UseLock: UseLock{Bits: 0x0},

			VolatileKeyPermission: VolatileKeyPermission{Bits: 0x0},

			SecureBoot: SecureBoot{Bits1: 0x3, Bits2: 0xf7},

			KdfIvLoc:     0x0,
			KdfIvStr:     [2]uint8{0x69, 0x76},
			Reserved68:   [9]uint8{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0},
			UserExtra:    0x0,
			UserExtraAdd: 0x0,
			LockValue:    0x55,
			LockConfig:   0x55,
			SlotLocked:   0xffff,
			ChipOptions:  ChipOptions{Bits1: 0xe, Bits2: 0x60},
			X509Format: [4]X509Format{
				{Bits: 0x0},
				{Bits: 0x0},
				{Bits: 0x0},
				{Bits: 0x0}},
			KeyConfig: [16]KeyConfig{
				{Bits1: 0x53, Bits2: 0x0},
				{Bits1: 0x53, Bits2: 0x0},
				{Bits1: 0x73, Bits2: 0x0},
				{Bits1: 0x73, Bits2: 0x0},
				{Bits1: 0x73, Bits2: 0x0},
				{Bits1: 0x38, Bits2: 0x0},
				{Bits1: 0x7c, Bits2: 0x0},
				{Bits1: 0x1c, Bits2: 0x0},
				{Bits1: 0x3c, Bits2: 0x0},
				{Bits1: 0x1a, Bits2: 0x0},
				{Bits1: 0x3c, Bits2: 0x0},
				{Bits1: 0x30, Bits2: 0x0},
				{Bits1: 0x3c, Bits2: 0x0},
				{Bits1: 0x30, Bits2: 0x0},
				{Bits1: 0x12, Bits2: 0x0},
				{Bits1: 0x30, Bits2: 0x0},
			}}
		got Config608
	)
	if err := Unmarshal(golden608, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf(" got: %v", got)
		t.Errorf("want: %v", want)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	want, err := os.ReadFile("testdata/608.json")
	if err != nil {
		t.Fatal(err)
	}
	var c Config608
	if err := Unmarshal(golden608, &c); err != nil {
		t.Fatal(err)
	}

	got, err := json.MarshalIndent(c, "", " ")
	if err != nil {
		t.Fatal(err)
	}

	if *flagWriteTestdata {
		if err := os.WriteFile("testdata/608.json", got, 0644); err != nil {
			t.Fatal(err)
		}
		want = got
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf(" got: %s", string(got))
		t.Errorf("want: %s", string(want))
	}
}

func TestMarshal(t *testing.T) {
	c := Config608{
		I2CAddress: 0x6a,
		Reserved17: 0,
		CountMatch: CountMatch{Bits: 0},
		ChipMode:   ChipMode608{Bits: 1},
		SlotConfig: [16]SlotConfig{
			{Bits1: 0x85, Bits2: 0x00},
			{Bits1: 0x82, Bits2: 0x00},
		},
	}

	b, err := Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	got := b[16 : 16+8]
	want := Default608[:8]

	if !bytes.Equal(got, want) {
		t.Errorf(" got: %s", hex.Dump(got))
		t.Errorf("want: %s", hex.Dump(want))
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	c := DefaultConfig608()

	b, err := Marshal(c)
	if err != nil {
		t.Fatal(err)
	}

	got := b[16:]
	want := Default608

	if !bytes.Equal(got, want) {
		t.Errorf(" got: %s", hex.Dump(got))
		t.Errorf("want: %s", hex.Dump(want))
	}
}

func TestUnmarshalPartial(t *testing.T) {
	var (
		want = Config608{
			I2CAddress: 0x6a,
			Reserved17: 0,
			CountMatch: CountMatch{Bits: 0},
			ChipMode:   ChipMode608{Bits: 1},
			SlotConfig: [16]SlotConfig{
				{Bits1: 0x85, Bits2: 0x00},
				{Bits1: 0x82, Bits2: 0x00},
			},
		}
		got Config608
	)
	if err := UnmarshalPartial(Default608[:8], 16, &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf(" got: %v", got)
		t.Errorf("want: %v", want)
	}
}
