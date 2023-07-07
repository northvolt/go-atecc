package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/northvolt/go-atecc"
	"github.com/northvolt/go-atecc/pkg/ateccconf"
	"github.com/peterbourgon/ff/v3/ffcli"
)

const (
	inputDefault = "default"
	inputHex     = "hex"
	inputJSON    = "json"
	inputDevice  = "device"

	outputGo     = "go"
	outputHex    = "hex"
	outputJSON   = "json"
	outputDevice = "device"
)

var allOutputs = []string{outputHex, outputJSON, outputDevice}

type confConfig struct {
	rootConfig *rootConfig
	in         io.Reader
	out        io.Writer
	err        io.Writer
	input      string
	output     string
	dry        bool
	json       bool
	genKeys    bool
	newAddr    string
}

func (c *confConfig) Exec(ctx context.Context, _ []string) error {
	if c.rootConfig.verbose {
		fmt.Fprintf(c.err, "config\n")
	}

	// Only connect to device when needed. This allow you to convert
	// configurations between differenct formats.
	var (
		dev  *atecc.Dev
		conf ateccconf.Config608
		info *deviceInfo
	)
	if c.input == inputDevice || c.output == outputDevice {
		if d, bus, err := newATECC(ctx, c.rootConfig); err != nil {
			return err
		} else {
			dev = d
			defer bus.Close()
		}

		if di, err := getDeviceInfo(ctx, dev); err != nil {
			return err
		} else {
			info = di
		}

		// Parse configuration from the device config zone
		if err := ateccconf.Unmarshal(info.ConfigZone, &conf); err != nil {
			return err
		}
	}

	provisionConf, err := createProvisionConfig(c.input, c.in, conf)
	if err != nil {
		return err
	}

	var i2cAddr uint16
	if c.newAddr != "" {
		if a, err := getI2CAddress(c.newAddr, c.rootConfig.trustPlatformFormat); err != nil {
			return err
		} else {
			i2cAddr = a
		}
	}

	err = useProvisionConfig(
		ctx, c.dry, c.output, c.out, i2cAddr, info, &conf, provisionConf, dev,
	)
	if err != nil {
		return err
	}

	if c.genKeys && info.IsDataZoneLocked {
		fmt.Fprintln(c.out, "Generating New Keys")
		if err := keyGen(ctx, c.out, c.dry, dev); err != nil {
			return err
		}
	}

	return nil
}

func useProvisionConfig(
	ctx context.Context, dry bool, output string, w io.Writer, i2cAddr uint16,
	di *deviceInfo, deviceConf *ateccconf.Config608,
	provisionConf *ateccconf.Config608, d *atecc.Dev,
) error {
	provisionBytes, err := ateccconf.Marshal(provisionConf)
	if err != nil {
		return err
	}

	// Change the I²C address when requested
	if i2cAddr != 0 {
		provisionBytes[ateccconf.PermanentOffset608] = byte(i2cAddr << 1)
		provisionConf.I2CAddress = byte(i2cAddr << 1)
	}

	switch output {
	case outputHex:
		fmt.Fprintln(w, prettyHexIndent(provisionBytes[ateccconf.PermanentOffset608:], "", " "))
		return nil
	case outputGo:
		conf := provisionBytes[ateccconf.PermanentOffset608:]

		var src strings.Builder
		src.WriteString("[...]byte{")
		for i, b := range conf {
			if (i % 8) == 0 {
				src.WriteString("\n ")
			}
			fmt.Fprintf(&src, " 0x%02x,", b)
		}
		src.WriteString("\n}")
		fmt.Fprintln(w, src.String())
		return nil
	case outputJSON:
		return writeJSON(w, provisionConf)
	case outputDevice:
		fmt.Fprintln(w, "Serial number:")
		fmt.Fprintln(w, prettyHex(di.SerialNumber))

		fmt.Fprintln(w, "Current I2C Address:")
		fmt.Fprintln(w, prettyHex([]byte{deviceConf.I2CAddress}))
		fmt.Fprintln(w, "Provision I2C Address:")
		fmt.Fprintln(w, prettyHex([]byte{provisionConf.I2CAddress}))

		if dry {
			fmt.Fprintln(w, "Configuration:")
			fmt.Fprintln(w, prettyHex(provisionBytes[ateccconf.PermanentOffset608:]))
			fmt.Fprintln(w, `
WARNING! This operation is irreversible! Once you lock the configuration to the
device, you will not be able to change it.

To continue with this operation, re-run with -dry=false.`)
			return nil
		}

		if !di.IsConfigZoneLocked {
			// TODO: create manifest
			// https://github.com/MicrochipTech/cryptoauth_trustplatform_designsuite/blob/master/docs/TrustPlatform_manifest_file_format_2019-09-26_A.pdf
			// https://github.com/MicrochipTech/cryptoauth_trustplatform_designsuite/tree/master/assets/python/manifest_helper
			if err := d.WriteConfigZone(ctx, provisionBytes); err != nil {
				return err
			}

			// Verify config zone
			currentBytes, err := d.ReadConfigZone(ctx)
			if err != nil {
				return err
			}

			// Skip the permanent manufacture specific header
			if !bytes.Equal(
				currentBytes[ateccconf.PermanentOffset608:],
				provisionBytes[ateccconf.PermanentOffset608:],
			) {
				return fmt.Errorf("configuration read from device does not match")
			}

			if err := d.LockConfigZone(ctx); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(w, "    Locked, skipping")
		}

		println("\nActivating Configuration")
		if !di.IsDataZoneLocked {
			if err := keyGen(ctx, w, dry, d); err != nil {
				return err
			}
			if err := d.LockDataZone(ctx); err != nil {
				return err
			}
		} else {
			fmt.Fprintln(w, "    Already active")
		}
		return nil
	default:
		outputs := strings.Join(allOutputs, ", ")
		return fmt.Errorf("atecc: valid outputs are %s", outputs)
	}
}

func keyGen(ctx context.Context, w io.Writer, dry bool, d *atecc.Dev) error {
	// Read latest config zone after writes and all
	configZone, err := d.ReadConfigZone(ctx)
	if err != nil {
		return err
	}

	var conf ateccconf.Config608
	if err := ateccconf.Unmarshal(configZone, &conf); err != nil {
		return err
	}

	printSkipMsg := func(slot int, msg string) {
		fmt.Fprintf(w, "    Skipping key pair generation in slot %d: %s\n", slot, msg)
	}

	for i := 0; i < 16; i++ {
		if !conf.KeyConfig[i].Private() {
			continue
		}

		// Data zone is already locked, additional conditions apply
		if conf.LockValue.IsLocked() {
			if !conf.SlotConfig[i].WriteConfig().GenKeyEnabled {
				printSkipMsg(i, "GenKey is disabled")
				continue
			}
			if conf.SlotLocked.IsLocked(i) {
				printSkipMsg(i, "Slot has been locked")
				continue
			}
			if conf.KeyConfig[i].RequireAuth() {
				printSkipMsg(i, "Slot requires authorization")
				continue
			}
			if conf.KeyConfig[i].PersistentDisable() {
				printSkipMsg(i, "Slot requires persistent latch")
				continue
			}
		}

		if dry {
			printSkipMsg(i, "Re-run with -dry=false to generate new key")
			continue
		}

		fmt.Fprintln(w, "    Generating key pair in slot", i)
		pub, err := d.GenerateKey(ctx, uint8(i))
		if err != nil {
			return err
		}

		if p, err := pemEncodePublicKey(pub); err != nil {
			return err
		} else {
			fmt.Fprintln(w, p)
		}
	}

	return nil
}

// createProvisionConfig creates a configuration for provisioning a device.
func createProvisionConfig(provision string, r io.Reader, deviceConf ateccconf.Config608) (*ateccconf.Config608, error) {
	switch provision {
	case inputDefault:
		return ateccconf.DefaultConfig608(), nil
	case inputHex:
		in, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}

		// Remove any whitespace incl newline
		s := strings.Join(strings.Fields(string(in)), "")

		b, err := hex.DecodeString(s)
		if err != nil {
			return nil, err
		}

		var conf ateccconf.Config608
		err = ateccconf.UnmarshalPartial(b, ateccconf.PermanentOffset608, &conf)
		return &conf, err
	case inputJSON:
		var conf ateccconf.Config608
		err := json.NewDecoder(r).Decode(&conf)
		return &conf, err
	case inputDevice:
		return &deviceConf, nil
	default:
		return nil, fmt.Errorf("valid config sources are default, device, hex, json")
	}
}

func newConfCmd(
	rootConfig *rootConfig, in io.Reader, out io.Writer, err io.Writer,
) *ffcli.Command {
	cfg := confConfig{
		rootConfig: rootConfig,
		in:         in,
		out:        out,
		err:        err,
	}

	fs := flag.NewFlagSet("atecc config", flag.ExitOnError)
	fs.StringVar(&cfg.input, "input", inputDefault, "Use this input for creating the provisioning configuration of the device: default (built-in), hex (stdin), json (stdin), device (read from device)")
	fs.StringVar(&cfg.output, "output", outputHex, "Use this output for the provisioning configuration: go, hex, json, device (write to device)")
	fs.BoolVar(&cfg.dry, "dry", true, "When disabled, data will be committed to device (this is irreversible!)")
	fs.BoolVar(&cfg.json, "json", false, "Use JSON format")
	fs.StringVar(&cfg.newAddr, "new-addr", "", "Change I2C address to this")
	fs.BoolVar(&cfg.genKeys, "gen", false, "Generate new keys")
	rootConfig.registerFlags(fs)

	return addLongHelp(&ffcli.Command{
		Name:       "config",
		ShortUsage: "config",
		ShortHelp:  "Writes a general purpose configuration to test the hardware.",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	})
}
