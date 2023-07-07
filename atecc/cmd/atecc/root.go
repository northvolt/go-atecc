package main

import (
	"context"
	"flag"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type rootConfig struct {
	verbose             bool
	iface               string
	bus                 int
	addr                string
	trustPlatformFormat bool
	devIndex            int
	// devInterface        string
	devIdentity string
}

func (c *rootConfig) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.verbose, "v", false, "increase log verbosity")
	fs.StringVar(&c.iface, "i", "i2c", "interface type, hid or i2c")
	fs.IntVar(&c.bus, "bus", 0, "i2c bus to use")
	fs.StringVar(&c.addr, "addr", "", "i2c address in hex")
	// fs.StringVar(&c.devInterface, "dev-interface", "auto", "dev kit interface type")
	// TODO: fallback to i2c address (change that to empty string) when empty
	fs.IntVar(&c.devIndex, "dev-index", 0, "device index when enumerating")
	fs.StringVar(&c.devIdentity, "dev-identity", "", "device identity is the I2C address or the bus number for the SWI interface device")
	fs.BoolVar(&c.trustPlatformFormat, "trust-platform-format", false, "use cryptoauthlib trust platform format instead of default common format")
}

func (c *rootConfig) Exec(context.Context, []string) error {
	return flag.ErrHelp
}

func newRootCmd() (*ffcli.Command, *rootConfig) {
	var cfg rootConfig

	fs := flag.NewFlagSet("atecc", flag.ExitOnError)
	cfg.registerFlags(fs)

	return addLongHelp(&ffcli.Command{
		Name:       "atecc",
		ShortUsage: "atecc [flags] <subcommand>",
		ShortHelp:  "Utilities to start developing and using your ATECC device.",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}), &cfg
}

var ateccLongHelp = `

GENERAL
If you use one of the dev kits with multiple secure elements, specify the device
identity to choose a specific element. Specify it similar to a I²C address or
use one of the common names for the configurations:

  TNGTLS     0x35 (0x6a)
  TFLXTLS    0x36 (0x6c)
  MAHDA      0x60 (0xc0)`
