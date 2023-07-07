package atecc

import "testing"

func TestParseKitDevice(t *testing.T) {
	buf := []byte("ECC608B TWI 00(6C)")

	dev, err := parseKitDevice(buf)
	if err != nil {
		t.Fatal(err)
	}
	if dev.DeviceType != DeviceATECC608 {
		t.Errorf("%v != %v", dev.DeviceType, DeviceATECC608)
	}
	if dev.KitType != KitTypeI2C {
		t.Errorf("%v != %v", dev.KitType, KitTypeI2C)
	}
	if dev.Address != 0x6c {
		t.Errorf("x%0x != x%0x", dev.Address, 0x6c)
	}
}
