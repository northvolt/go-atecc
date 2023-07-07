/*
atecc is a tool to communicates with the hardware security module.

It supports ATECC608A and communicates using I²C.
*/
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
)

func main() {
	var (
		in  = os.Stdin
		out = os.Stdout
		err = os.Stderr
	)

	rootCmd, cfg := newRootCmd()
	rootCmd.Subcommands = []*ffcli.Command{
		newConfCmd(cfg, in, out, err),
		newInfoCmd(cfg, out, err),
		newRandCmd(cfg, out, err),
		newSignCmd(cfg, in, out, err),
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		var num = 0
		for range c {
			num += 1
			if num >= 3 {
				os.Exit(1)
			} else {
				cancel()
			}
		}
	}()

	if err := rootCmd.ParseAndRun(ctx, os.Args[1:]); err != nil {
		if !errors.Is(err, context.Canceled) {
			libPrefix := "atecc: "
			msg := strings.TrimPrefix(err.Error(), libPrefix)
			fmt.Fprintf(os.Stderr, "%s: %s\n", rootCmd.Name, msg)
			os.Exit(1)
		} else if cfg.verbose {
			fmt.Fprintf(os.Stderr, "%s: cancelled\n", rootCmd.Name)
		}
	}
}
