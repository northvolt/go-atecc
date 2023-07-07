package atecc

import (
	"errors"
)

// Protocol errors. See datasheet for specification.
var (
	errCheckMacVerifyFailed = errors.New("atecc: check mac verify failed")

	// errParseError is used when protocol was not understood.
	//
	// Received length, op-code or any parameter was illegal.
	errParseError = errors.New("atecc: protocol error")

	errProcessFailure = errors.New("atecc: ecc failed to process")
	errSelfTestFailed = errors.New("atecc: self-test failed")
	errHealthTest     = errors.New("atecc: health test failed")
	errExecution      = errors.New("atecc: execution error")

	// errWakeSuccessful is used when device is successfully woken up.
	//
	// This is an error for any command except for wake.
	errWakeSuccessful = errors.New("atecc: wake successful")

	// errCRC is used for checksum missmatch or other communication error.
	//
	// Bad CRC, command not properly received by device or other error.
	//
	// This is a transient error and the command should be re-transmitted.
	errCRC = errors.New("atecc: crc or communication error")

	errUnknown = errors.New("atecc: unknown error")
)

// validateResponseStatusCode validates the status code returned by protocol.
//
// The status code is the first byte of the response and indicates how the
// command was processed by the device.
func validateResponseStatusCode(response []byte) error {
	if len(response) == 0 {
		return errors.New("atecc: empty response")
	}

	statusCode := response[0]
	switch statusCode {
	case 0x00:
		return nil
	case 0x01:
		return errCheckMacVerifyFailed
	case 0x03:
		return errParseError
	case 0x05:
		return errProcessFailure
	case 0x07:
		return errSelfTestFailed
	case 0x08:
		return errHealthTest
	case 0x0f:
		return errExecution
	case 0x11:
		return errWakeSuccessful
	case 0xff:
		return errCRC
	default:
		return errUnknown
	}
}

// Package errors.
var (
	errRecvBuffer = errors.New("atecc: recv buffer too small")
)
