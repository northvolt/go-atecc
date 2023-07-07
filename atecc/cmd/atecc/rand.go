package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
)

type randConfig struct {
	rootConfig *rootConfig
	out        io.Writer
	err        io.Writer
	bytes      int64
	timeout    time.Duration
}

func (c *randConfig) Exec(ctx context.Context, _ []string) error {
	if c.rootConfig.verbose {
		fmt.Fprintln(c.err, "random")
	}

	d, bus, err := newATECC(ctx, c.rootConfig)
	if err != nil {
		return err
	}
	defer bus.Close()

	if c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var written int64
	r := d.Random(ctx)
	if c.bytes > 0 {
		written, err = io.CopyN(c.out, r, c.bytes)
	} else {
		written, err = io.Copy(c.out, r)
	}
	if err != nil {
		return err
	}
	if c.rootConfig.verbose {
		fmt.Fprintln(c.err, "wrote", written)
	}

	return nil
}

func newRandCmd(rootConfig *rootConfig, out io.Writer, err io.Writer) *ffcli.Command {
	cfg := randConfig{
		rootConfig: rootConfig,
		out:        out,
		err:        err,
	}

	fs := flag.NewFlagSet("atecc random", flag.ExitOnError)
	fs.Int64Var(&cfg.bytes, "bytes", 0, "maximum bytes to read")
	fs.DurationVar(&cfg.timeout, "timeout", 0, "maximum time to read eg 1s, 500ms")
	rootConfig.registerFlags(fs)

	return &ffcli.Command{
		Name:       "random",
		ShortUsage: "random",
		ShortHelp:  "Reads random bytes from device and outputs on stdout.",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	}
}
