package atecc

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/northvolt/go-atecc/pkg/ateccconf"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/crypto/cryptobyte/asn1"
)

type deviceState int

const (
	deviceStateUnknown deviceState = iota
	deviceStateIdle
	deviceStateActive
)

// Zone is a configuration zone.
type Zone uint8

// Configuration zones.
const (
	ZoneConfig Zone = 0x00
	ZoneOTP    Zone = 0x01
	ZoneData   Zone = 0x02
)

const (
	zoneSizeConfig = 128
	zoneSizeOTP    = 64
)

func getZoneSize(zone Zone, slot uint16) (int, error) {
	switch zone {
	case ZoneConfig:
		return zoneSizeConfig, nil
	case ZoneOTP:
		return zoneSizeOTP, nil
	case ZoneData:
		if slot < 8 {
			return 36, nil
		} else if slot == 8 {
			return 416, nil
		} else if slot < 16 {
			return 72, nil
		} else {
			return 0, errors.New("atecc: invalid slot received")
		}
	default:
		return 0, errors.New("atecc: invalid zone received")
	}
}

type Dev struct {
	hal   HAL
	state deviceState
	cfg   IfaceConfig
	enc   packetEncoder
	log   Logger

	clockDivider ateccconf.ClockDivider
}

// New returns a new ATECC device using the supplied HAL for communication.
func New(ctx context.Context, hal HAL, cfg IfaceConfig) (*Dev, error) {
	// TODO: make this call into NewI2C etc based on device type?
	d := &Dev{
		hal:   hal,
		state: deviceStateUnknown,
		cfg:   cfg,
		log:   getLogger(cfg),
	}
	d.hal = &halDebug{"ecc", getLogger(cfg), d.hal}
	return d, d.init(ctx)
}

func (d *Dev) init(ctx context.Context) error {
	var buf [1]byte
	_, err := d.readBytesZone(ctx, ZoneConfig, 0, ateccconf.ChipModeOffset, buf[:])
	if err != nil {
		return err
	}

	var conf ateccconf.Config608
	err = ateccconf.UnmarshalPartial(buf[:], ateccconf.ChipModeOffset, &conf)
	if err != nil {
		return err
	}

	d.clockDivider = conf.ChipMode.ClockDivider()
	return nil
}

// Revision gets the device revision.
//
// This information is hard coded into the device. Use it to determine the
// version of the device.
func (d *Dev) Revision(ctx context.Context) ([]byte, error) {
	var recv [4]byte
	p, err := newInfoCommand(infoModeRevision)
	if err != nil {
		return nil, err
	}
	n, err := d.executeResponse(ctx, p, recv[:])
	return recv[:n], err
}

// Random returns a random reader.
//
// The underlying reader reads 32 byte random data from the device at a time.
//
// Use io.ReadFull to fill a buffer.
func (d *Dev) Random(ctx context.Context) io.Reader {
	return &randReader{ctx, d}
}

// SerialNumber returns the serial number of the device.
//
// The returned serial number will be 9 bytes.
func (d *Dev) SerialNumber(ctx context.Context) ([]byte, error) {
	return d.serialNumber(ctx)
}

func (d *Dev) ReadZone(ctx context.Context, zone Zone, slot uint16, block uint8, offset uint8, b []byte) (int, error) {
	return d.readZone(ctx, zone, slot, block, offset, b)
}

// ReadConfigZone reads the complete device configuration zone.
func (d *Dev) ReadConfigZone(ctx context.Context) ([]byte, error) {
	var buf [128]byte
	n, err := d.readConfigZone(ctx, buf[:])
	return buf[:n], err
}

// IsConfigZoneLocked returns true if the configuration zone is locked.
//
// This is the same as calling IsLocked(ctx, ZoneConfig).
func (d *Dev) IsConfigZoneLocked(ctx context.Context) (bool, error) {
	return d.IsLocked(ctx, ZoneConfig)
}

// IsDataZoneLocked returns true if the data zone is locked.
//
// This is the same as calling IsLocked(ctx, ZoneData).
func (d *Dev) IsDataZoneLocked(ctx context.Context) (bool, error) {
	return d.IsLocked(ctx, ZoneData)
}

func (d *Dev) IsLocked(ctx context.Context, zone Zone) (bool, error) {
	var buf [atcaWordSize]byte

	// Read the word with the lock bytes
	// (UserExtra, Selector, LockValue, LockConfig)
	const block = ateccconf.LockOffsetBlock
	const offset = ateccconf.LockOffsetWord
	if _, err := d.readZone(ctx, ZoneConfig, 0, block, offset, buf[:]); err != nil {
		return false, err
	}

	var conf ateccconf.Config608
	err := ateccconf.UnmarshalPartial(buf[:], ateccconf.LockOffset, &conf)
	if err != nil {
		return false, err
	}

	switch zone {
	case ZoneConfig:
		return conf.LockConfig.IsLocked(), nil
	case ZoneData:
		return conf.LockValue.IsLocked(), nil
	default:
		return false, errors.New("atecc: unknown lock zone")
	}
}

func (d *Dev) LockConfigZone(ctx context.Context) error {
	return d.lockConfigZone(ctx)
}

func (d *Dev) LockDataZone(ctx context.Context) error {
	return d.lockDataZone(ctx)
}

func (d *Dev) LockDataSlot(ctx context.Context, slot uint8) error {
	return d.lockDataSlot(ctx, slot)
}

// GenerateKey generates a new random private key in slot/handle.
func (d *Dev) GenerateKey(ctx context.Context, slot uint8) (crypto.PublicKey, error) {
	var pk [64]byte
	n, err := d.generateKey(ctx, slot, pk[:])
	if err != nil {
		return nil, err
	}

	if n != 64 {
		return nil, errors.New("atecc: unexpected public key size: " + strconv.Itoa(n))
	}
	var x, y big.Int
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x.SetBytes(pk[:32]),
		Y:     y.SetBytes(pk[32:]),
	}, nil
}

// PublicKey returns the public key in the specific slot.
func (d *Dev) PublicKey(ctx context.Context, slot uint8) (crypto.PublicKey, error) {
	var pk [64]byte
	n, err := d.publicKey(ctx, slot, pk[:])
	if err != nil {
		return nil, err
	}

	if n != 64 {
		return nil, errors.New("atecc: unexpected public key size: " + strconv.Itoa(n))
	}
	var x, y big.Int
	return &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x.SetBytes(pk[:32]),
		Y:     y.SetBytes(pk[32:]),
	}, nil
}

// Sign signs the message using the private key in the specified slot.
//
// This function executes the sign command to sign a 32-byte external message
// using the private key in the specified slot. It returns the ASN.1 encoded
// signature.
func (d *Dev) Sign(ctx context.Context, key int, msg []byte) ([]byte, error) {
	var sig [64]byte
	n, err := d.sign(ctx, uint16(key), msg, sig[:])
	if err != nil {
		return nil, err
	} else if n != 64 {
		return nil, fmt.Errorf("atecc: unexpected signature size: %d", n)
	}

	var r, s big.Int
	var b cryptobyte.Builder
	b.AddASN1(asn1.SEQUENCE, func(b *cryptobyte.Builder) {
		b.AddASN1BigInt(r.SetBytes(sig[:32]))
		b.AddASN1BigInt(s.SetBytes(sig[32:]))
	})
	return b.Bytes()
}

func (d *Dev) PrivateKey(ctx context.Context, key uint8) (crypto.PrivateKey, error) {
	// TODO: consistent type for key
	pub, err := d.PublicKey(ctx, key)
	if err != nil {
		return nil, err
	}
	return &privateKey{ctx, pub, d, key}, nil
}

// VerifyExtern verifies a signature using external input.
//
// The signature provided is expected to be in ASN.1 format.
func (d *Dev) VerifyExtern(ctx context.Context, msg, sig []byte, pub crypto.PublicKey) (bool, error) {
	var (
		r, s  = big.Int{}, big.Int{}
		inner cryptobyte.String
	)
	input := cryptobyte.String(sig)
	if !input.ReadASN1(&inner, asn1.SEQUENCE) ||
		!input.Empty() ||
		!inner.ReadASN1Integer(&r) ||
		!inner.ReadASN1Integer(&s) ||
		!inner.Empty() {
		return false, errors.New("atecc: invalid signature")
	}
	var signature [64]byte
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])

	var pk [64]byte
	switch pub := pub.(type) {
	case *ecdsa.PublicKey:
		pub.X.FillBytes(pk[:32])
		pub.Y.FillBytes(pk[32:])
	default:
		return false, errors.New("atecc: unsupported public key")
	}

	return d.verifyExtern(ctx, msg, signature[:], pk[:])
}

// WriteBytesZone writes the data into the config, OTP or data zone.
//
// If ZoneConfig is unlocked, it may be written to. If ZoneData is unlocked,
// 32-byte writes are allowed to slots and OTP.
//
// Offset and length must be multiples of 32 or the write will fail.
func (d *Dev) WriteBytesZone(ctx context.Context, zone Zone, slot uint16, offset uint8, data []byte) error {
	_, err := d.writeBytesZone(ctx, zone, slot, offset, data)
	return err
}

// WriteConfigZone writes the data into the config zone.
//
// This method works similar to how WriteBytesZone work except that it also
// writes the UserExtraData if all other data was written successfully.
//
// Warning: if UserExtraData or UserExtraDataAdd is not 0x55 ('U'), these
// values will be permanent and the corresponding zones will be locked. If so,
// this is irreversible!
func (d *Dev) WriteConfigZone(ctx context.Context, data []byte) error {
	_, err := d.writeConfigZone(ctx, data)
	return err
}

// execute executes the command and returns any error encountered.
func (d *Dev) execute(ctx context.Context, p *packet) error {
	var buf [1]byte
	_, err := d.executeResponse(ctx, p, buf[:])
	return err
}

// executeResponse executes the command and returns bytes written and error.
//
// The command is encoded and transfered to the device. It returns the number
// of bytes read into recv together with any error encountered.
func (d *Dev) executeResponse(ctx context.Context, p *packet, recv []byte) (int, error) {
	b, err := d.enc.Encode(p)
	if err != nil {
		return 0, err
	}

	// send the command to the device
	for i := -1; i < d.cfg.RxRetries; i++ {
		if d.state != deviceStateActive {
			if err = d.hal.Wake(); err == nil {
				d.state = deviceStateActive
			}
		}

		if _, err = d.hal.Write(b); err == nil {
			d.state = deviceStateActive
			break
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(d.cfg.WakeDelay):
		}
	}
	if err != nil {
		return 0, err
	}

	// Put device back into idle mode once finished. This function is called even
	// if we would encounter a panic.
	defer func() {
		_ = d.hal.Idle()
		d.state = deviceStateIdle
	}()

	// wait for the operation to finish
	t, err := getExecutionTime(d.cfg.DeviceType, d.clockDivider, p.opcode)
	if err != nil {
		return 0, err
	}
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case <-time.After(t):
	}

	// make room for 1 byte size and 2 byte crc
	buf := make([]byte, len(recv)+3)
	size, err := d.hal.Read(buf[:])
	if err != nil {
		if errors.Is(err, errRecvBuffer) {
			fmt.Fprintf(os.Stderr, "atecc: receive buffer overflowed\n")
			debug.PrintStack()
		}

		return 0, err
	}

	if len(buf) == 0 {
		return 0, errors.New("atecc: no response")
	} else if len(buf) < 4 {
		return 0, errors.New("atecc: receive failed")
	}

	// response is 1 byte size, payload and 2 bytes crc
	sizedResponse, crc := buf[0:size-2], buf[size-2:]
	if crc16(sizedResponse) != binary.LittleEndian.Uint16(crc) {
		return 0, errors.New("atecc: received crc missmatch")
	}

	// error responses are always 4 bytes long
	if size == 4 {
		if err = validateResponseStatusCode(sizedResponse[1:]); err != nil {
			if d.log != nullLogger {
				d.log.Printf("invalid status code: %v\n", err)
				d.log.Printf("%s", string(debug.Stack()))
			}
			return 0, err
		}
	}

	return copy(recv, sizedResponse[1:]), nil
}

type randReader struct {
	ctx context.Context
	d   *Dev
}

func (r *randReader) Read(b []byte) (int, error) {
	return r.d.random(r.ctx, b)
}

// privateKey wraps an atecc device and key slot for private cryptography.
//
// privateKey implements crypto.Signer and crypto.PrivateKey.
type privateKey struct {
	ctx context.Context
	p   crypto.PublicKey
	d   *Dev
	key uint8
}

var _ crypto.Signer = &privateKey{}

// Public returns the public key corresponding to the opaque, private key.
//
// This implements crypto.Signer.
func (priv *privateKey) Public() crypto.PublicKey {
	return priv.p
}

// Sign signs digest with the private key.
//
// This implements crypto.Signer.
func (priv *privateKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return priv.d.Sign(priv.ctx, int(priv.key), digest)
}
