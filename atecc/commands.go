package atecc

import "errors"

// General device command opcodes
//nolint unused commands
const (
	atcaCheckMac    = 0x28 // CheckMac command op-code
	atcaDeriveKey   = 0x1c // DeriveKey command op-code
	atcaInfo        = 0x30 // Info command op-code
	atcaGenDig      = 0x15 // GenDig command op-code
	atcaGenKey      = 0x40 // GenKey command op-code
	atcaHMAC        = 0x11 // HMAC command op-code
	atcaLock        = 0x17 // Lock command op-code
	atcaMAC         = 0x08 // MAC command op-code
	atcaNonce       = 0x16 // Nonce command op-code
	atcaPause       = 0x01 // Pause command op-code
	atcaPrivWrite   = 0x46 // PrivWrite command op-code
	atcaRandom      = 0x1b // Random command op-code
	atcaRead        = 0x02 // Read command op-code
	atcaSign        = 0x41 // Sign command op-code
	atcaUpdateExtra = 0x20 // UpdateExtra command op-code
	atcaVerify      = 0x45 // GenKey command op-code
	atcaWrite       = 0x12 // Write command op-code
	atcaECDH        = 0x43 // ECDH command op-code
	atcaCounter     = 0x24 // Counter command op-code
	atcaDelete      = 0x13 // Delete command op-code
	atcaSHA         = 0x47 // SHA command op-code
	atcaAES         = 0x51 // AES command op-code
	atcaKDF         = 0x56 // KDF command op-code
	atcaSecureBoot  = 0x80 // Secure Boot command op-code
	atcaSelfTest    = 0x77 // Self test command op-code
)

type infoMode uint8

const (
	infoModeRevision infoMode = 0x0
)

func newInfoCommand(mode infoMode) (*packet, error) {
	return newPacket(atcaInfo, uint8(mode), 0, nil)
}

type lockZone uint8

const (
	lockZoneConfig   = lockZone(0x00)
	lockZoneData     = lockZone(0x01)
	lockZoneDataSlot = lockZone(0x02)
)

type lockMode uint8

const (
	lockModeNoCRC = lockMode(0x80)
)

func newLockCommand(zone lockZone, mode lockMode, crc uint16) (*packet, error) {
	return newPacket(atcaLock, uint8(zone)|uint8(mode), crc, nil)
}

func newReadCommand(zone Zone, param2 uint16, block bool) (*packet, error) {
	param1 := uint8(zone)
	if block {
		param1 = param1 | atcaZoneReadWrite32
	}
	return newPacket(atcaRead, param1, param2, nil)
}

//nolint unused
const (
	genKeyModePrivate      = 0x04 // generate private key
	genKeyModePublic       = 0x00 // calculate public key
	genKeyModeDigest       = 0x08 // key digest
	genKeyModePubKeyDigest = 0x10 // public key digest
	genKeyModeMAC          = 0x20 // calculate MAC of public key + session key
)

func newGenKeyCommand(mode uint8, keyId uint8, otherData []byte) (*packet, error) {
	return newPacket(atcaGenKey, mode, uint16(keyId), otherData)
}

type randomMode uint8

//nolint unused
const (
	randomModeUpdateSeed   randomMode = 0x0
	randomModeNoUpdateSeed randomMode = 0x01
)

func newRandomCommand(mode randomMode) (*packet, error) {
	return newPacket(atcaRandom, uint8(mode), 0x0, nil)
}

type nonceTarget uint8

//nolint unused
const (
	nonceTargetTempKey   nonceTarget = 0x0  // TempKey
	nonceTargetMsgDigBuf nonceTarget = 0x40 // Message Digest Buffer (ATECC608)
	nonceTargetAltKeyBuf nonceTarget = 0x80 // Alternate Key Buffer
)

type nonceMode uint8

// Nonce modes.
const (
	nonceModeSeedUpdate nonceMode = 0x00 // update seed
	// nonceModeNoSeedUpdate nonceMode = 0x01 // do not update seed
	nonceModePassthrough nonceMode = 0x03 // pass-through
)

// Nonce mode flags.
//nolint unused
const (
	nonceModeFlagInputLenMask uint8 = 0x20 // Nonce mode: input size mask
	nonceModeFlagInputLen32   uint8 = 0x00 // Nonce mode: input size is 32 bytes
	nonceModeFlagInputLen64   uint8 = 0x20 // Nonce mode: input size is 64 bytes
)

func newNonceCommand(mode nonceMode, target nonceTarget, param2 uint16, keyIn []byte) (*packet, error) {
	param1 := uint8(mode)
	if mode == nonceModePassthrough {
		if len(keyIn) == 32 {
			param1 = param1 | nonceModeFlagInputLen32
		} else if len(keyIn) == 64 {
			param1 = param1 | nonceModeFlagInputLen64
		} else {
			return nil, errors.New("atecc: invalid nonce size requested")
		}
	} else {
		if len(keyIn) != 20 {
			return nil, errors.New("atecc: invalid nonce size requested")
		}
	}

	param1 = param1 | uint8(target)
	return newPacket(atcaNonce, param1, param2, keyIn)
}

type signMode uint8

//nolint unused
const (
	signModeInternal   signMode = 0x00 // Sign mode	 0: internal
	signModeInvalidate signMode = 0x01 // Sign mode bit 1: Signature will be used for Verify(Invalidate)
	signModeIncludeSN  signMode = 0x40 // Sign mode bit 6: include serial number
	signModeExternal   signMode = 0x80 // Sign mode bit 7: external
)

type signSource uint8

const (
	signSourceTempKey   signSource = 0x00 // Sign mode message source is TempKey
	signSourceMsgDigBuf signSource = 0x20 // Sign mode message source is the Message Digest Buffer
)

func newSignCommand(mode signMode, source signSource, keyId uint16) (*packet, error) {
	return newPacket(atcaSign, uint8(mode)|uint8(source), keyId, nil)
}

type verifyMode uint8

// Verify modes.
//nolint unused
const (
	verifyModeStored           verifyMode = 0x00 // stored
	verifyModeValidateExternal verifyMode = 0x01 // validate external
	verifyModeExternal         verifyMode = 0x02 // external
	verifyModeValidate         verifyMode = 0x03 // validate
	verifyModeInvalidate       verifyMode = 0x07 // invalidate
)

// Verify key types.
//nolint unused
const (
	verifyKeyB283 = 0x0000 // B283
	verifyKeyK283 = 0x0001 // K283
	verifyKeyP256 = 0x0004 // P256
)

type verifySource uint8

const (
	verifySourceTempKey   verifySource = 0x00 // TempKey
	verifySourceMsgDigBuf verifySource = 0x20 // Message Digest Buffer (ATECC608)
)

func newVerifyCommand(mode verifyMode, source verifySource, keyId uint16, sig, pub, otherData []byte) (*packet, error) {
	var data [atcaCmdSizeMax]byte

	n := copy(data[:], sig)
	if mode == verifyModeExternal {
		var pubSize int
		switch keyId {
		case verifyKeyP256:
			pubSize = 64
		default:
			return nil, errors.New("atecc: unsupported verify key size")
		}
		if pubSize != len(pub) {
			return nil, errors.New("atecc: invalid public key received")
		}
		n += copy(data[n:], pub)
	} else if otherData != nil {
		n += copy(data[n:], otherData)
	}

	return newPacket(atcaVerify, uint8(mode)|uint8(source), keyId, data[:n])
}

func newWriteCommand(zone Zone, addr uint16, value []byte, mac []byte) (*packet, error) {
	var data [atcaBlockSize * 2]byte

	param1 := uint8(zone)
	n := copy(data[:], value)
	if n == atcaWordSize {
		if mac != nil {
			return nil, errors.New("atecc: unexpected mac for word write")
		}
	} else if n == atcaBlockSize {
		param1 = param1 | atcaZoneReadWrite32
		if mac != nil {
			copy(data[n:], mac)
		}
	} else {
		return nil, errors.New("atecc: write data exceeds block size")
	}

	return newPacket(atcaWrite, param1, addr, data[:])
}

type updateMode uint8

const (
	updateModeUserExtra    updateMode = 0x00
	updateModeUserExtraAdd updateMode = 0x01
)

func newUpdateExtraCommand(mode updateMode, newValue byte) (*packet, error) {
	return newPacket(atcaUpdateExtra, uint8(mode), uint16(newValue), nil)
}
