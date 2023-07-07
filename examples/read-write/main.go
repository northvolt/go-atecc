package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"log"
	"os"

	"github.com/northvolt/go-atecc/atecc"
	"github.com/northvolt/go-atecc/pkg/ateccconf"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/host/v3"
)

type ReadWriteConfig struct {
	Clear uint16
}

var readWriteConfig = map[atecc.DeviceType]ReadWriteConfig{
	atecc.DeviceATECC608: {8},
}

func main() {
	ctx := context.Background()

	if _, err := host.Init(); err != nil {
		panic(err)
	}

	for i, ref := range i2creg.All() {
		println(i, ref.Name, ref.Aliases, ref.Number)
	}

	bus, err := i2creg.Open("")
	if err != nil {
		panic(err)
	}
	defer bus.Close()

	cfg := atecc.ConfigATECCX08A_I2CDefault(bus)
	cfg.Debug = log.New(os.Stderr, "", 0)
	cfg.I2C.Address = 0x60
	d, err := atecc.NewI2CDev(ctx, cfg)
	if err != nil {
		panic(err)
	}

	info, err := d.Revision(ctx)
	if err != nil {
		panic(err)
	}
	deviceType, err := atecc.DeviceTypeFromInfo(info)
	if err != nil {
		panic(err)
	}
	slots, ok := readWriteConfig[deviceType]
	if !ok {
		panic("unsupported device type")
	}

	configData, err := d.ReadConfigZone(ctx)
	if err != nil {
		panic(err)
	}

	var config ateccconf.Config608
	if err := ateccconf.Unmarshal(configData, &config); err != nil {
		panic(err)
	}

	var (
		writeData [32]byte
		readData  [32]byte
	)

	println("Generating data using RAND command")
	var rr = d.Random(ctx)
	if _, err = io.ReadFull(rr, writeData[:]); err != nil {
		panic(err)
	}
	println("    Generated data:")
	println(hex.Dump(writeData[:]))

	// Writing a data to slot
	println("Write command:")
	println("    Writing data to slot", slots.Clear)
	err = d.WriteBytesZone(ctx, atecc.ZoneData, slots.Clear, 0, writeData[:])
	if err != nil {
		panic(err)
	}
	println("    Write Success")

	// Reading the data in the clear from slot
	println("Read command:")
	println("    Reading data stored in slot", slots.Clear)
	if _, err := d.ReadZone(ctx, atecc.ZoneData, slots.Clear, 0, 0, readData[:]); err != nil {
		panic(err)
	}
	println("    Read data:")
	println(hex.Dump(readData[:]))

	// Compare the read data to the written data
	println("Verifing read data matches written data:")
	if bytes.Equal(readData[:], writeData[:]) {
		println("    Data Matches")
	} else {
		println("    Data Does Not Matches")
	}

	// In the Python code, there's an example where data is written encrypted.
	//
	// These functions doesn't seem to belong to cryptoauthlib and a lot of the
	// actual encryption is done on the host.
}
