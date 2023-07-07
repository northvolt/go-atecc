package atecc

import (
	"context"
	"errors"

	"github.com/northvolt/go-atecc/ateccconf"
)

// Definitions for Zone and Address Parameters
const (
	// atcaZoneReadWrite32 is the zone bit 7 set: access 32 bytes, otherwise 4 bytes.
	atcaZoneReadWrite32 = 0x80
)

func (d *Dev) lockConfigZone(ctx context.Context) error {
	return d.lock(ctx, lockZoneConfig, lockModeNoCRC, 0)
}

func (d *Dev) lockDataZone(ctx context.Context) error {
	return d.lock(ctx, lockZoneData, lockModeNoCRC, 0)
}

func (d *Dev) lockDataSlot(ctx context.Context, slot uint8) error {
	return d.lock(ctx, lockZoneDataSlot, lockMode(slot<<2), 0)
}

func (d *Dev) lock(ctx context.Context, zone lockZone, mode lockMode, crc uint16) error {
	command, err := newLockCommand(zone, mode, crc)
	if err != nil {
		return err
	}

	return d.execute(ctx, command)
}

// TODO: rewrite in idiomatic go
func (d *Dev) readBytesZone(ctx context.Context, zone Zone, slot uint16, offset int, data []byte) (int, error) {
	var buf [atcaBlockSize]byte
	var dataIdx = 0
	var curOffset = 0

	// Always succeed reading 0 bytes
	if len(data) == 0 {
		return 0, nil
	}

	zoneSize, err := getZoneSize(zone, 0)
	if err != nil {
		return 0, err
	}

	// make sure we don't read past end of zone
	if offset+len(data) > zoneSize {
		return 0, errors.New("atecc: invalid parameter recieved")
	}

	readBuf := buf[:atcaBlockSize]
	curBlock := offset / atcaBlockSize
	for dataIdx < len(data) {
		// Read word size when we have less than a block left to read
		if len(readBuf) == atcaBlockSize && zoneSize-curBlock*atcaBlockSize < atcaBlockSize {
			readBuf = buf[:atcaWordSize]
			curOffset = ((dataIdx + offset) / atcaWordSize) % (atcaBlockSize / atcaWordSize)
		}

		n, err := d.readZone(ctx, zone, slot, uint8(curBlock), uint8(curOffset), readBuf)
		if err != nil {
			return dataIdx, err
		}

		readBufIdx := 0
		readOffset := curBlock*atcaBlockSize + curOffset*atcaWordSize
		// Check if read data starts before the requested chunk
		if readOffset < offset {
			readBufIdx = offset - readOffset
		}

		// Calculate how much data from the read buffer we want to copy
		copyLength := n - readBufIdx
		if len(data)-dataIdx < copyLength {
			copyLength = len(data) - dataIdx
		}

		copy(data[dataIdx:], readBuf[readBufIdx:readBufIdx+copyLength])
		dataIdx += copyLength
		if n == atcaBlockSize {
			curBlock += 1
		} else {
			curOffset += 1
		}
	}
	return dataIdx, nil
}

func (d *Dev) readZone(ctx context.Context, zone Zone, slot uint16, block uint8, offset uint8, data []byte) (int, error) {
	if len(data) != atcaBlockSize && len(data) != atcaWordSize {
		return 0, errors.New("atecc: invalid read zone size")
	}

	addr, err := getAddr(zone, slot, block, offset)
	if err != nil {
		return 0, err
	}

	// build a read command
	blockMode := len(data) == atcaBlockSize
	cmd, err := newReadCommand(zone, addr, blockMode)
	if err != nil {
		return 0, err
	}

	return d.executeResponse(ctx, cmd, data)
}

func (d *Dev) readConfigZone(ctx context.Context, data []byte) (int, error) {
	return d.readBytesZone(ctx, ZoneConfig, 0, 0x00, data)
}

// getAddr computes the address given the zone, slot, block, and offset.
func getAddr(zone Zone, slot uint16, block uint8, offset uint8) (uint16, error) {
	var addr uint16

	// Mask the offset
	offset = offset & 0x07

	switch zone {
	case ZoneConfig:
		fallthrough
	case ZoneOTP:
		addr = uint16(block) << 3
		addr = addr | uint16(offset)
		return addr, nil
	case ZoneData:
		addr = slot << 3
		addr = addr | uint16(offset)
		addr = addr | (uint16(block) << 8)
		return addr, nil
	default:
		return 0, errors.New("atecc: invalid zone received")
	}
}

// serialNumber reads the config and extracts the 9 bytes serial number.
func (d *Dev) serialNumber(ctx context.Context) ([]byte, error) {
	var buf [atcaBlockSize]byte
	_, err := d.readZone(ctx, ZoneConfig, 0, 0, 0, buf[:])
	if err != nil {
		return nil, err
	}

	var conf ateccconf.Config608
	err = ateccconf.UnmarshalPartial(buf[:], 0, &conf)
	if err != nil {
		return nil, err
	}

	var serialNumber [9]byte
	copy(serialNumber[:], conf.SN03[:])
	copy(serialNumber[4:], conf.SN48[:])
	return serialNumber[:], nil
}

func (d *Dev) generateKey(ctx context.Context, keyId uint8, publicKey []byte) (int, error) {
	return d.genKeyBase(ctx, genKeyModePrivate, keyId, nil, publicKey)
}

func (d *Dev) publicKey(ctx context.Context, keyId uint8, publicKey []byte) (int, error) {
	return d.genKeyBase(ctx, genKeyModePublic, keyId, nil, publicKey)
}

// getKeyBase issues the GenKey command which does various things.
//
// This function generates and executes the GenKey command, which generate a
// private key, compute a public key and/or compute a digest of a public key.
func (d *Dev) genKeyBase(ctx context.Context, mode uint8, keyId uint8, otherData []byte, publicKey []byte) (int, error) {
	command, err := newGenKeyCommand(mode, keyId, otherData)
	if err != nil {
		return 0, err
	}

	var recv [64]byte
	n, err := d.executeResponse(ctx, command, recv[:])
	if err != nil {
		return 0, err
	}

	if publicKey != nil {
		return copy(publicKey, recv[:n]), nil
	} else {
		return 0, nil
	}
}

// sign signs the message using the private key in the specified slot.
//
// This function executes the sign command to sign a 32-byte external message
// using the private key in the specified slot.
//
// The message to be signed will be loaded into the Message Digest Buffer to
// the ATECC608 device or TempKey for other devices.
//
// Signature format is R and S integers in big-endian format. 64 bytes for P256
// curve.
func (d *Dev) sign(ctx context.Context, keyId uint16, msg []byte, sig []byte) (int, error) {
	// make sure RNG has updated its seed
	if _, err := d.random(ctx, nil); err != nil {
		return 0, err
	}

	var (
		target = nonceTargetTempKey
		source = signSourceTempKey
	)
	if d.cfg.DeviceType == DeviceATECC608 {
		target = nonceTargetMsgDigBuf
		source = signSourceMsgDigBuf
	}

	if err := d.nonceLoad(ctx, target, msg); err != nil {
		return 0, err
	}
	return d.signBase(ctx, signModeExternal, source, keyId, sig)
}

// verifyExtern verifies a signature using external input.
//
// Executes the Verify command, which verifies a signature (ECDSA verify
// operation) with all components (message, signature, and public key)
// supplied.
//
// The message to be signed will be loaded into the Message Digest Buffer to
// the ATECC608 device or TempKey for other devices.
func (d *Dev) verifyExtern(ctx context.Context, msg, sig, pub []byte) (bool, error) {
	if msg == nil || sig == nil || pub == nil {
		return false, errors.New("atcab: expected message, signature and public key")
	}

	var (
		target = nonceTargetTempKey
		source = verifySourceTempKey
	)
	if d.cfg.DeviceType == DeviceATECC608 {
		target = nonceTargetMsgDigBuf
		source = verifySourceMsgDigBuf
	}

	if err := d.nonceLoad(ctx, target, msg); err != nil {
		return false, err
	}

	command, err := newVerifyCommand(
		verifyModeExternal, source, verifyKeyP256, sig, pub, nil,
	)
	if err != nil {
		return false, err
	}
	err = d.execute(ctx, command)
	ok := err != errCRC
	return ok, err
}

// SignBase executes the Sign command, which generates a signature using the
// ECDSA algorithm.
func (d *Dev) signBase(ctx context.Context, mode signMode, source signSource, keyId uint16, sig []byte) (int, error) {
	if sig == nil {
		return 0, errors.New("atecc: signature buffer was nil")
	}

	command, err := newSignCommand(mode, source, keyId)
	if err != nil {
		return 0, nil
	}

	return d.executeResponse(ctx, command, sig)
}

// random executes the random command, which generates a 32 byte random number.
func (d *Dev) random(ctx context.Context, dst []byte) (int, error) {
	command, err := newRandomCommand(randomModeUpdateSeed)
	if err != nil {
		return 0, err
	}

	var recv [32]byte
	n, err := d.executeResponse(ctx, command, recv[:])
	if err != nil {
		return 0, err
	} else if n != 32 {
		return 0, errors.New("atecc: unexpected random response size")
	}

	if dst != nil {
		return copy(dst, recv[:]), nil
	} else {
		return 0, nil
	}
}

func (d *Dev) nonceLoad(ctx context.Context, target nonceTarget, numIn []byte) error {
	if numIn == nil {
		return errors.New("atecc: requires input for nonce")
	}

	command, err := newNonceCommand(nonceModePassthrough, target, 0, numIn)
	if err != nil {
		return err
	}
	return d.execute(ctx, command)
}

// nonceRand generates a random nonce.
//
// The random nonce is generated by combining a host nonce and a device random
// number.
//
//lint:ignore U1000 unused function
func (d *Dev) nonceRand(ctx context.Context, numIn []byte, rand []byte) (int, error) {
	if numIn == nil {
		return 0, errors.New("atecc: requires input for nonce")
	}

	command, err := newNonceCommand(nonceModeSeedUpdate, nonceTargetTempKey, 0, numIn)
	if err != nil {
		return 0, err
	}
	return d.executeResponse(ctx, command, rand)
}

func (d *Dev) write(ctx context.Context, zone Zone, addr uint16, data []byte, mac []byte) error {
	command, err := newWriteCommand(zone, addr, data, mac)
	if err != nil {
		return err
	}

	return d.execute(ctx, command)
}

func (d *Dev) writeZone(ctx context.Context, zone Zone, slot uint16, block uint8, offset uint8, data []byte) error {
	if len(data) != atcaBlockSize && len(data) != atcaWordSize {
		return errors.New("atecc: invalid write zone size")
	}

	// The get address function checks the remaining variables
	addr, err := getAddr(zone, slot, block, offset)
	if err != nil {
		return err
	}

	if err := d.write(ctx, zone, addr, data, nil); err != nil {
		return err
	}

	return nil
}

// TODO: rewrite in idiomatic go
func (d *Dev) writeBytesZone(ctx context.Context, zone Zone, slot uint16, offset uint8, data []byte) (int, error) {
	if zone == ZoneData && slot > 15 {
		return 0, errors.New("atecc: invalid slot")
	}

	// Always succeed reading 0 bytes
	if len(data) == 0 {
		return 0, nil
	}

	// TODO: docs says it should be a multiple of 32?
	if offset%atcaWordSize != 0 {
		return 0, errors.New("atecc: invalid offset")
	}
	if len(data)%atcaWordSize != 0 {
		return 0, errors.New("atecc: invalid length")
	}

	zoneSize, err := getZoneSize(zone, slot)
	if err != nil {
		return 0, err
	}
	if int(offset)+len(data) > zoneSize {
		return 0, errors.New("atecc: invalid offset and zone")
	}

	block := offset / atcaBlockSize
	word := (offset % atcaBlockSize) / atcaWordSize

	var index = 0
	for index < len(data) {
		// Makes sure we skip writing to the selector, user extra, and lock bytes.
		// These need to be written using the UpdateExtra command.
		inLockBlock := zone == ZoneConfig && block == ateccconf.LockOffsetBlock
		inLockWord := inLockBlock && word == ateccconf.LockOffsetWord

		// Write block-wise when we're aligned and there's a full block available.
		remaining := len(data) - index
		writeBlock := word == 0 && remaining >= atcaBlockSize

		if writeBlock && !inLockBlock {
			err = d.writeZone(ctx, zone, slot, block, 0, data[index:index+atcaBlockSize])
			if err != nil {
				return index, err
			}
			index += atcaBlockSize
			block += 1
		} else {
			if !inLockWord {
				err = d.writeZone(ctx, zone, slot, block, word, data[index:index+atcaWordSize])
				if err != nil {
					return index, err
				}
			}
			index += atcaWordSize
			word += 1
			if word == atcaBlockSize/atcaWordSize {
				block += 1
				word = 0
			}
		}
	}
	return index, nil
}

// TODO: rewrite in idiomatic go
func (d *Dev) writeConfigZone(ctx context.Context, data []byte) (int, error) {
	// Be very strict about the size. We don't want anyone to accidentally miss
	// that this function actually skips the first 16 bytes, which is unexpected.
	if zoneSizeConfig != len(data) {
		return 0, errors.New("atecc: config data size mismatch")
	}

	// Write config zone excluding UserExtra and Selector
	const offset = ateccconf.PermanentOffset608
	n, err := d.writeBytesZone(ctx, ZoneConfig, 0, offset, data[offset:])
	if err != nil {
		return n, err
	}

	// Write the UserExtra and UserExtraAdd. This may fail if either value is
	// already non-zero.
	if err := d.updateExtra(ctx, updateModeUserExtra, data[84]); err != nil {
		return n, err
	}
	return n, d.updateExtra(ctx, updateModeUserExtraAdd, data[85])
}

// updateExtra updates the two extra bytes within the configuration zone.
//
// This function executes the UpdateExtra command to update the values of the
// extra bytes within the configuration zone (bytes 84 and 85).
func (d *Dev) updateExtra(ctx context.Context, mode updateMode, newValue byte) error {
	command, err := newUpdateExtraCommand(mode, newValue)
	if err != nil {
		return err
	}

	return d.execute(ctx, command)
}
