package atecc

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

type halKit struct {
	phy HAL
	buf []byte
	cfg IfaceConfig
}

var errNoDevice = errors.New("atecc: no device found")

func newHALKit(ctx context.Context, phy HAL, cfg IfaceConfig) (*halKit, error) {
	buf := make([]byte, getPacketSize(cfg))
	phy = &halDebug{"kit", getLogger(cfg), phy}
	kit := &halKit{phy, buf, cfg}
	return kit, kit.init(ctx)
}

func kitIdFromDeviceType(deviceType DeviceType) string {
	switch deviceType {
	case DeviceATECC608:
		return "ECC608"
	default:
		return "unknown"
	}
}

func deviceTypeFromKitId(id string) (DeviceType, error) {
	if strings.HasPrefix(id, "ECC6") {
		return DeviceATECC608, nil
	} else {
		return DeviceType(0), errors.New("atecc: unknown device type")
	}
}

func kitTypeFromKitIface(iface string) (KitType, error) {
	switch iface {
	case "TWI":
		return KitTypeI2C, nil
	case "SWI":
		return KitTypeSWI, nil
	case "SPI":
		return KitTypeSPI, nil
	default:
		return KitType(0), errors.New("atecc: unknown kit type")
	}
}

func kitIface(kitType KitType) string {
	switch kitType {
	case KitTypeI2C:
		return "i2c"
	case KitTypeSWI:
		return "swi"
	case KitTypeSPI:
		return "spi"
	default:
		return "unknown"
	}
}

const (
	kitMaxScanCount = 8
	kitMaxTxBuf     = 32

	// kitTxWrapSize = 10
	kitMsgSize    = 32
	kitRxWrapSize = kitMsgSize + 6
)

func (h *halKit) init(ctx context.Context) error {
	var (
		devIndex    int
		kitType     KitType
		devIdentity uint8
	)
	switch h.cfg.IfaceType {
	case IfaceHID:
		devIndex = h.cfg.HID.DevIndex
		kitType = h.cfg.HID.KitType
		devIdentity = h.cfg.HID.DevIdentity
	default:
		kitType = KitTypeAuto
	}

	// Iterate to find the target device
	for i := 0; i < kitMaxScanCount; i++ {
		dev, err := h.getKitDeviceByIndex(i)
		if errors.Is(err, errNoDevice) {
			continue
		} else if err != nil {
			return err
		}

		// Check if the returned device is a device we want to pick
		if devIndex != 0 && devIndex != i {
			continue
		}
		if devIdentity != 0 && devIdentity != dev.Address {
			continue
		}
		if h.cfg.DeviceType != dev.DeviceType {
			continue
		}
		if kitType != KitTypeAuto && kitType != dev.KitType {
			continue
		}

		if kitType != KitTypeAuto {
			if err := h.selectInterface(kitType); err != nil {
				return err
			}
		}

		return h.selectDevice(dev.Address)
	}

	return errors.New("atecc: failed to discover device")
}

func (h *halKit) Wake() error {
	kitId := kitIdFromDeviceType(h.cfg.DeviceType)
	command := fmt.Sprintf("%c:w()\n", kitId[0])

	var data [10]byte
	n, err := h.executeResponse([]byte(command), data[:])
	if err != nil {
		return err
	} else {
		return checkWakeUp(data[:n])
	}
}

func (h *halKit) Idle() error {
	kitId := kitIdFromDeviceType(h.cfg.DeviceType)
	command := fmt.Sprintf("%c:i()\n", kitId[0])
	return h.execute([]byte(command))
}

func (h *halKit) Write(data []byte) (int, error) {
	kitId := kitIdFromDeviceType(h.cfg.DeviceType)
	payload := strings.ToUpper(hex.EncodeToString(data))
	command := fmt.Sprintf("%c:t(%s)\n", kitId[0], payload)
	return h.phySend([]byte(command))
}

func (h *halKit) Read(dst []byte) (int, error) {
	msg := hex.EncodedLen(len(dst)) + kitRxWrapSize
	pkt := h.cfg.HID.PacketSize
	buf := make([]byte, (msg/pkt+1)*pkt)

	n, err := h.phyRecv(buf)
	if err != nil {
		return 0, err
	}

	return kitParseRsp(buf[:n], dst)
}

func (h *halKit) execute(command []byte) error {
	var data [10]byte
	_, err := h.executeResponse(command, data[:])
	return err
}

func (h *halKit) executeResponse(command []byte, data []byte) (int, error) {
	if _, err := h.phySend([]byte(command)); err != nil {
		return 0, err
	}

	n, err := h.phyRecv(h.buf)
	if err != nil {
		return 0, err
	}
	return kitParseRsp(h.buf[:n], data)
}

func (h *halKit) getKitDeviceByIndex(index int) (kitDevice, error) {
	command := fmt.Sprintf("board:device(%02X)\n", index)
	if _, err := h.phySend([]byte(command)); err != nil {
		return kitDevice{}, err
	}

	if n, err := h.phyRecv(h.buf); err != nil {
		return kitDevice{}, err
	} else {
		return parseKitDevice(h.buf[:n])
	}
}

func (h *halKit) selectInterface(kitType KitType) error {
	kitId := kitIdFromDeviceType(h.cfg.DeviceType)
	command := fmt.Sprintf(
		"%c:physical:interface(%02X)\n", kitId[0], kitIface(kitType),
	)
	return h.execute([]byte(command))
}

func (h *halKit) selectDevice(address uint8) error {
	kitId := kitIdFromDeviceType(h.cfg.DeviceType)
	command := fmt.Sprintf(
		"%c:physical:select(%02X)\n", kitId[0], address,
	)
	return h.execute([]byte(command))
}

type kitDevice struct {
	DeviceType DeviceType
	KitType    KitType
	Address    uint8
}

func parseKitDevice(buf []byte) (kitDevice, error) {
	var (
		kitId    string
		kitIface string
		index    uint8
		address  uint8
	)
	if bytes.HasPrefix(buf, []byte("no_device")) {
		return kitDevice{}, errNoDevice
	}
	_, err := fmt.Sscanf(
		string(buf), "%s %s %02X(%02X)", &kitId, &kitIface, &index, &address,
	)
	if err != nil {
		return kitDevice{}, fmt.Errorf("atecc: invalid kit device: %w", err)
	}

	if dt, err := deviceTypeFromKitId(kitId); err != nil {
		return kitDevice{}, err
	} else if kt, err := kitTypeFromKitIface(kitIface); err != nil {
		return kitDevice{}, err
	} else {
		return kitDevice{dt, kt, address}, nil
	}
}

func (h *halKit) phySend(txData []byte) (int, error) {
	left := len(txData)
	sent := 0
	for left > 0 {
		n := copy(h.buf, txData[sent:])
		for ; n < cap(h.buf); n++ {
			h.buf[n] = 0
		}

		n, err := h.phy.Write(h.buf)
		if err != nil {
			return sent, err
		}

		left -= n
		sent += n
	}

	return sent, nil
}

func (h *halKit) phyRecv(data []byte) (int, error) {
	left := len(data)
	read := 0
	for left > 0 {
		n, err := h.phy.Read(h.buf)
		if err != nil {
			return read, err
		}

		// end early on response end
		if index := bytes.IndexByte(h.buf, '\n'); index != -1 {
			copy(data[read:], h.buf[:index]) // ignore return for overflow check below
			read += index
			break
		}

		copy(data[read:], h.buf) // ignore return for overflow check below
		read += n
		left -= n
	}

	// error out to make sure we never loose any data
	if read > cap(data) {
		return read, errors.New("atecc: buffer overflow")
	}

	return read, nil
}

func kitParseRsp(reply []byte, dst []byte) (int, error) {
	var status [1]byte
	n, err := hex.Decode(status[:], reply[0:2])
	if err != nil {
		return 0, err
	} else if err := validateResponseStatusCode(status[:n]); err != nil {
		return 0, err
	}

	index := bytes.IndexByte(reply[3:], ')')
	if index == -1 {
		return 0, errors.New("atecc: failed to find end of frame")
	}
	size := hex.DecodedLen(index)
	if size > cap(dst) {
		return 0, errRecvBuffer
	}

	body := reply[3 : 3+index]
	return hex.Decode(dst, body)
}

func getPacketSize(cfg IfaceConfig) int {
	switch cfg.IfaceType {
	case IfaceHID:
		return cfg.HID.PacketSize
	default:
		panic("atecc: unsupported iface type")
	}
}
