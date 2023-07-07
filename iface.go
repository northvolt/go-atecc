package atecc

import (
	"time"

	"periph.io/x/conn/v3/i2c"
)

type IfaceType int

const (
	IfaceI2C IfaceType = iota
	IfaceHID
)

// IfaceConfig is the configuration object for a device.
//
// Logical device configurations describe the device type and logical
// interface.
type IfaceConfig struct {
	// IfaceType affects how communication with the device is done.
	IfaceType IfaceType
	// DeviceType affects how communication with the device is done.
	DeviceType DeviceType
	// I2C contains I²C specific configuration.
	I2C I2CConfig
	// HID contains HID specific configuration.
	HID HIDConfig
	// WakeDelay defines the time to wait for the device before waking up.
	//
	// This represents the tWHI + tWLO and is configured based on device type.
	WakeDelay time.Duration
	// RxRetries is the number of retries to attempt when receiving data.
	RxRetries int
	// Debug is used for debug output.
	Debug Logger
}

type I2CConfig struct {
	Address uint16
	Bus     i2c.Bus
}

type KitType int

const (
	KitTypeAuto KitType = iota
	KitTypeI2C
	KitTypeSWI
	KitTypeSPI
)

type HIDConfig struct {
	// DevIndex is the HID enumeration index to use unless DevIdentity is set.
	DevIndex int

	// KitType indicates the underlying interface to use.
	//
	// This is known as dev_interface in cryptoauthlib.
	KitType KitType

	// DevIdentity is the identity of the device.
	//
	// For I²C, this is the I²C target address. For the SWI interface, this is
	// the bus number.
	DevIdentity uint8

	// VendorID of the kit.
	VendorID uint16

	// ProductID of the kit.
	ProductID uint16

	// PacketSize is the size of the USB packet.
	PacketSize int
}

// ConfigATECCX08A_I2CDefault returns a default config for an ECCx08A device.
//
// TODO: re-think where we put bus, who owns it (who closes, do we have Close?)
func ConfigATECCX08A_I2CDefault(bus i2c.Bus) IfaceConfig {
	return IfaceConfig{
		IfaceType:  IfaceI2C,
		DeviceType: DeviceATECC608,
		WakeDelay:  1500 * time.Microsecond,
		RxRetries:  20,
		I2C: I2CConfig{
			Address: 0x60,
			Bus:     bus,
		},
	}
}

const (
	vendorAtmel = 0x03eb

	productTrustPlatform = 0x2312
)

// ConfigATECCX08A_KitHIDDefault returns a configuration for the Kit protocol.
func ConfigATECCX08A_KitHIDDefault() IfaceConfig {
	return IfaceConfig{
		IfaceType:  IfaceHID,
		DeviceType: DeviceATECC608,
		HID: HIDConfig{
			DevIndex:    0,
			KitType:     KitTypeAuto,
			DevIdentity: 0,
			VendorID:    vendorAtmel,
			ProductID:   productTrustPlatform,
			PacketSize:  64,
		},
	}
}
