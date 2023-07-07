package atecc

import (
	"errors"
	"time"

	"github.com/northvolt/go-atecc/ateccconf"
)

// DeviceType represents a physical device type.
type DeviceType int

const (
	DeviceATECC608 DeviceType = iota
)

func (dt DeviceType) String() string {
	switch dt {
	case DeviceATECC608:
		return "ATECC608"
	default:
		return "unknown"
	}
}

// DeviceTypeFromInfo returns the device type based on the info byte array.
func DeviceTypeFromInfo(revision []byte) (DeviceType, error) {
	if len(revision) < 3 {
		return 0, errors.New("atecc: device type revision too small")
	}
	switch revision[2] {
	case 0x60:
		return DeviceATECC608, nil
	default:
		return 0, errors.New("attec: unknown device revision")
	}
}

const (
	deviceExecutionTime608M0 = iota
	deviceExecutionTime608M1
	deviceExecutionTime608M2
)

// deviceExecutionTimes holds execution times for device supported commands.
var deviceExecutionTimes = []map[uint8]time.Duration{
	// ATECC608-M0
	{
		atcaAES:         27 * time.Millisecond,
		atcaCheckMac:    40 * time.Millisecond,
		atcaCounter:     25 * time.Millisecond,
		atcaDeriveKey:   50 * time.Millisecond,
		atcaECDH:        75 * time.Millisecond,
		atcaGenDig:      25 * time.Millisecond,
		atcaGenKey:      115 * time.Millisecond,
		atcaInfo:        5 * time.Millisecond,
		atcaKDF:         165 * time.Millisecond,
		atcaLock:        35 * time.Millisecond,
		atcaMAC:         55 * time.Millisecond,
		atcaNonce:       20 * time.Millisecond,
		atcaPrivWrite:   50 * time.Millisecond,
		atcaRandom:      23 * time.Millisecond,
		atcaRead:        5 * time.Millisecond,
		atcaSecureBoot:  80 * time.Millisecond,
		atcaSelfTest:    250 * time.Millisecond,
		atcaSHA:         36 * time.Millisecond,
		atcaSign:        115 * time.Millisecond,
		atcaUpdateExtra: 10 * time.Millisecond,
		atcaVerify:      105 * time.Millisecond,
		atcaWrite:       45 * time.Millisecond,
	},

	// ATECC608-M1
	{
		atcaAES:         27 * time.Millisecond,
		atcaCheckMac:    40 * time.Millisecond,
		atcaCounter:     25 * time.Millisecond,
		atcaDeriveKey:   50 * time.Millisecond,
		atcaECDH:        172 * time.Millisecond,
		atcaGenDig:      35 * time.Millisecond,
		atcaGenKey:      215 * time.Millisecond,
		atcaInfo:        5 * time.Millisecond,
		atcaKDF:         165 * time.Millisecond,
		atcaLock:        35 * time.Millisecond,
		atcaMAC:         55 * time.Millisecond,
		atcaNonce:       20 * time.Millisecond,
		atcaPrivWrite:   50 * time.Millisecond,
		atcaRandom:      23 * time.Millisecond,
		atcaRead:        5 * time.Millisecond,
		atcaSecureBoot:  160 * time.Millisecond,
		atcaSelfTest:    625 * time.Millisecond,
		atcaSHA:         42 * time.Millisecond,
		atcaSign:        220 * time.Millisecond,
		atcaUpdateExtra: 10 * time.Millisecond,
		atcaVerify:      295 * time.Millisecond,
		atcaWrite:       45 * time.Millisecond,
	},
	// ATECC608-M2
	{
		atcaAES:         27 * time.Millisecond,
		atcaCheckMac:    40 * time.Millisecond,
		atcaCounter:     25 * time.Millisecond,
		atcaDeriveKey:   50 * time.Millisecond,
		atcaECDH:        531 * time.Millisecond,
		atcaGenDig:      35 * time.Millisecond,
		atcaGenKey:      653 * time.Millisecond,
		atcaInfo:        5 * time.Millisecond,
		atcaKDF:         165 * time.Millisecond,
		atcaLock:        35 * time.Millisecond,
		atcaMAC:         55 * time.Millisecond,
		atcaNonce:       20 * time.Millisecond,
		atcaPrivWrite:   50 * time.Millisecond,
		atcaRandom:      23 * time.Millisecond,
		atcaRead:        5 * time.Millisecond,
		atcaSecureBoot:  480 * time.Millisecond,
		atcaSelfTest:    2324 * time.Millisecond,
		atcaSHA:         75 * time.Millisecond,
		atcaSign:        665 * time.Millisecond,
		atcaUpdateExtra: 10 * time.Millisecond,
		atcaVerify:      1085 * time.Millisecond,
		atcaWrite:       45 * time.Millisecond,
	},
}

func getDeviceExecutionTime(dt DeviceType, div ateccconf.ClockDivider) (map[uint8]time.Duration, error) {
	switch dt {
	case DeviceATECC608:
		switch div {
		case ateccconf.ClockDividerM0:
			return deviceExecutionTimes[deviceExecutionTime608M0], nil
		case ateccconf.ClockDividerM1:
			return deviceExecutionTimes[deviceExecutionTime608M1], nil
		case ateccconf.ClockDividerM2:
			return deviceExecutionTimes[deviceExecutionTime608M2], nil
		default:
			return nil, errors.New("atecc: unknown clock divider")
		}
	default:
		return nil, errors.New("atecc: unknown execution time for device")
	}
}

func getExecutionTime(dt DeviceType, div ateccconf.ClockDivider, opcode uint8) (time.Duration, error) {
	executionTimes, err := getDeviceExecutionTime(dt, div)
	if err != nil {
		return 0, err
	}

	if t, ok := executionTimes[opcode]; !ok {
		return 0, errors.New("atecc: unknown execution time for op")
	} else {
		return t, nil
	}
}
