package main

import (
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/northvolt/go-atecc/atecc"
	"github.com/peterbourgon/ff/v3/ffcli"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

const (
	defaultI2CAddress     = 0x60
	defaultDeviceIdentity = 0
)

func newATECC(ctx context.Context, c *rootConfig) (*atecc.Dev, io.Closer, error) {
	switch c.iface {
	case "i2c":
		return newATECC_I2C(ctx, c)
	case "hid":
		return newATECC_HID(ctx, c)
	default:
		return nil, nil, errors.New("atecc: unknown interface")
	}
}

func newATECC_I2C(ctx context.Context, c *rootConfig) (*atecc.Dev, io.Closer, error) {
	i2cAddress, err := getI2CAddress(c.addr, c.trustPlatformFormat)
	if err != nil {
		return nil, nil, err
	}

	if _, err = host.Init(); err != nil {
		return nil, nil, err
	}
	bus, err := i2creg.Open(strconv.Itoa(c.bus))
	if err != nil {
		return nil, nil, fmt.Errorf("atecc: failed to connect to bus: %w", err)
	}

	cfg := atecc.ConfigATECCX08A_I2CDefault(bus)
	cfg.Debug = newLogger(c.verbose)
	cfg.I2C.Address = i2cAddress
	d, err := atecc.NewI2CDev(ctx, cfg)
	return d, bus, err
}

func newATECC_HID(ctx context.Context, c *rootConfig) (*atecc.Dev, io.Closer, error) {
	identity, err := getHIDDeviceIdentity(c.devIdentity, c.trustPlatformFormat)
	if err != nil {
		return nil, nil, err
	}

	cfg := atecc.ConfigATECCX08A_KitHIDDefault()
	cfg.Debug = newLogger(c.verbose)
	cfg.HID.DevIndex = c.devIndex
	cfg.HID.DevIdentity = identity

	return atecc.NewHIDDev(ctx, cfg)
}

func getI2CAddress(addrStr string, trustPlatformFormat bool) (uint16, error) {
	if addrStr == "" {
		return defaultI2CAddress, nil
	}
	addr, err := strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 16)
	if err != nil {
		return 0, err
	}

	if trustPlatformFormat {
		return uint16(addr >> 1), nil
	} else {
		return uint16(addr), nil
	}
}

func hidDeviceIdentityToString(idStr string) (uint16, error) {
	switch strings.ToUpper(idStr) {
	case "TNGTLS":
		return 0x35, nil
	case "TFLXTLS":
		return 0x36, nil
	case "MAHDA":
		return 0x60, nil
	default:
		return 0, errors.New("atecc: unknown HID device identity")
	}
}

func getHIDDeviceIdentity(idStr string, trustPlatformFormat bool) (uint8, error) {
	if idStr == "" {
		return defaultDeviceIdentity, nil
	}
	id, err := hidDeviceIdentityToString(idStr)
	if err != nil {
		id64, err := strconv.ParseUint(strings.TrimPrefix(idStr, "0x"), 16, 16)
		if err != nil {
			return 0, err
		}
		id = uint16(id64)
	}

	if trustPlatformFormat {
		return uint8(id), nil
	} else {
		return uint8(id << 1), nil
	}
}

func prettyHex(data []byte) string {
	return prettyHexIndent(data, "    ", "")
}

func prettyHexIndent(data []byte, prefix string, space string) string {
	var buf strings.Builder

	// prefix and space every 16 byte, and 2 hex, and one space/newline
	cols := 16
	size := (len(data)/cols+1)*(len(prefix)+len(space)+1) + len(data)*3
	buf.Grow(size)

	for i := range data {
		if i > 0 {
			switch i % cols {
			case 0:
				buf.WriteByte('\n')
			case cols / 2:
				buf.WriteByte(' ')
				buf.WriteString(space)
			default:
				buf.WriteByte(' ')
			}
		}
		if i%cols == 0 {
			buf.WriteString(prefix)
		}

		buf.WriteString(fmt.Sprintf("%02X", data[i:i+1]))
	}

	return buf.String()
}

func pemEncodePublicKey(pk crypto.PublicKey) (string, error) {
	der, err := x509.MarshalPKIXPublicKey(pk)
	if err != nil {
		return "", err
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	})), nil
}

func addLongHelp(cmd *ffcli.Command) *ffcli.Command {
	if cmd.LongHelp == "" {
		cmd.LongHelp = cmd.ShortHelp
	}

	cmd.LongHelp += ateccLongHelp

	return cmd
}

func newLogger(verbose bool) atecc.Logger {
	if verbose {
		return log.New(os.Stderr, "", 0)
	} else {
		return nil
	}
}
