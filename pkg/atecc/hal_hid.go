package atecc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/karalabe/usb"
)

// ErrUSBNotSupported is returned when the USB support is missing.
//
// When building, CGO is required for USB support. If CGO is not enabled, the
// HID interface will not be available.
var ErrUSBNotSupported = errors.New("atecc: usb support is missing")

// NewHIDDev returns an object that communicates over HID.
func NewHIDDev(ctx context.Context, cfg IfaceConfig) (*Dev, io.Closer, error) {
	if !usb.Supported() {
		return nil, nil, ErrUSBNotSupported
	}

	deviceInfos, err := usb.EnumerateHid(cfg.HID.VendorID, cfg.HID.ProductID)
	if err != nil {
		return nil, nil, fmt.Errorf("atecc: failed to get hid devices: %w", err)
	}
	for _, di := range deviceInfos {
		hid, e := di.Open()
		if e != nil {
			err = e
			continue
		}

		phy := newHALHID(hid, cfg)
		hal, err := newHALKit(ctx, phy, cfg)
		if err != nil {
			return nil, nil, err
		}
		var d *Dev
		if d, err = New(ctx, hal, cfg); err != nil {
			return nil, nil, err
		}
		return d, hid, err
	}
	if err != nil {
		return nil, nil, fmt.Errorf("atecc: %w", err)
	} else {
		return nil, nil, errors.New("atecc: no hid devices found")
	}
}

type halHID struct {
	usb usb.Device
	cfg IfaceConfig
}

func newHALHID(
	usb usb.Device,
	cfg IfaceConfig,
) *halHID {
	return &halHID{
		usb: usb,
		cfg: cfg,
	}
}

func (h *halHID) Write(p []byte) (int, error) {
	return h.usb.Write(p)
}

func (h *halHID) Read(p []byte) (int, error) {
	return h.usb.Read(p)
}

func (h *halHID) Idle() error {
	return errors.New("not implemented")
}

func (h *halHID) Wake() error {
	return errors.New("not implemented")
}
