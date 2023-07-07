package atecc

import (
	"encoding/binary"
	"errors"
)

// Command definitions
const (
	// atcaCmdSizeMin is the minimum size of a command.
	//
	// It includes opcode, size, param, param2 and crc.
	atcaCmdSizeMin uint8 = 7
	atcaCmdSizeMax       = 4*36 + 7
)

const (
	// atcaBlockSize is the size of a block
	atcaBlockSize = 32
	// atcaWordSize is the size of a word
	atcaWordSize = 4
)

// packet represents an ATCA packet
type packet struct {
	opcode uint8
	param1 uint8
	param2 uint16
	data   []byte
}

func newPacket(opcode uint8, param1 uint8, param2 uint16, data []byte) (*packet, error) {
	if len(data) > atcaCmdSizeMax {
		return nil, errors.New("atecc: data size exceeds maximum size")
	}
	return &packet{
		opcode: opcode,
		param1: param1,
		param2: param2,
		data:   data,
	}, nil
}

func (p *packet) Size() uint8 {
	return atcaCmdSizeMin + uint8(len(p.data))
}

// packetEncoder encodes packets.
type packetEncoder struct {
}

func (e *packetEncoder) Encode(p *packet) ([]byte, error) {
	size := p.Size()
	b := make([]byte, 0, size)
	b = append(b, size)
	b = append(b, p.opcode)
	b = append(b, p.param1)
	b = binary.LittleEndian.AppendUint16(b, p.param2)
	b = append(b, p.data...)
	return binary.LittleEndian.AppendUint16(b, crc16(b)), nil
}
