package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"text/template"

	"github.com/northvolt/go-atecc/pkg/atecc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type infoConfig struct {
	rootConfig *rootConfig
	out        io.Writer
	err        io.Writer
	json       bool
}

func (c *infoConfig) Exec(ctx context.Context, _ []string) error {
	if c.rootConfig.verbose {
		fmt.Fprintf(c.err, "info\n")
	}

	d, closer, err := newATECC(ctx, c.rootConfig)
	if err != nil {
		return err
	}
	defer closer.Close()

	di, err := getDeviceInfo(ctx, d)
	if err != nil {
		return err
	}

	if c.json {
		return writeJSON(c.out, di)
	} else {
		return writeText(c.out, di)
	}
}

const deviceInfoTemplate = `
Device Part:
    {{ .Name }}

Serial number:
{{ hex .SerialNumber }}

Configuration Zone:
{{ hex .ConfigZone }}

Check Device Locks
    Config Zone is {{ locked .IsConfigZoneLocked }}
    Data Zone is {{ locked .IsDataZoneLocked }}

{{ if .PublicKey -}}
{{ .PublicKey -}}
{{- end }}
Done
`

func writeText(w io.Writer, di *deviceInfo) error {
	funcs := template.FuncMap{
		"hex": prettyHex,
		"locked": func(b bool) string {
			if b {
				return "locked"
			} else {
				return "unlocked"
			}
		},
	}
	t, err := template.New("info").Funcs(funcs).Parse(deviceInfoTemplate)
	if err != nil {
		return err
	}

	return t.Execute(w, di)
}

func writeJSON(w io.Writer, data any) error {
	j, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return err
	}
	_, err = w.Write(j)
	return err
}

func newInfoCmd(
	rootConfig *rootConfig, out io.Writer, err io.Writer,
) *ffcli.Command {
	cfg := infoConfig{
		rootConfig: rootConfig,
		out:        out,
		err:        err,
	}

	fs := flag.NewFlagSet("atecc info", flag.ExitOnError)
	fs.BoolVar(&cfg.json, "json", false, "output in json mode")
	rootConfig.registerFlags(fs)

	return addLongHelp(&ffcli.Command{
		Name:       "info",
		ShortUsage: "info",
		ShortHelp:  "Returns information about the hardware security module.",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	})
}

type deviceInfo struct {
	Name               string `json:"name"`
	SerialNumber       []byte `json:"serial_number"`
	ConfigZone         []byte `json:"config_zone"`
	IsConfigZoneLocked bool   `json:"is_config_zone_locked"`
	IsDataZoneLocked   bool   `json:"is_data_zone_locked"`
	PublicKey          string `json:"public_key,omitempty"`
}

func getDeviceInfo(ctx context.Context, d *atecc.Dev) (*deviceInfo, error) {
	var di = &deviceInfo{}

	info, err := d.Revision(ctx)
	if err != nil {
		return nil, err
	}
	deviceType, err := atecc.DeviceTypeFromInfo(info)
	if err != nil {
		return di, err
	}
	di.Name = deviceType.String()

	di.SerialNumber, err = d.SerialNumber(ctx)
	if err != nil {
		return di, err
	}

	di.ConfigZone, err = d.ReadConfigZone(ctx)
	if err != nil {
		return di, err
	}
	di.IsConfigZoneLocked, err = d.IsLocked(ctx, atecc.ZoneConfig)
	if err != nil {
		return di, err
	}

	di.IsDataZoneLocked, err = d.IsLocked(ctx, atecc.ZoneData)
	if err != nil {
		return di, err
	}

	if di.IsDataZoneLocked {
		pk, err := d.PublicKey(ctx, 0)
		if err != nil {
			return nil, err
		}

		pub, err := pemEncodePublicKey(pk)
		if err != nil {
			return nil, err
		}

		di.PublicKey = pub
	}

	return di, nil
}
