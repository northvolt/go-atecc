package atecc

import (
	"encoding/hex"
	"strings"
)

// Logger is the interface used for debug messages.
//
// Some messages will be multiple lines.
type Logger interface {
	Printf(format string, args ...interface{})
}

type nullLoggerImpl struct{}

func (nullLoggerImpl) Printf(format string, args ...interface{}) {}

// nullLogger is a logger that does nothing.
var nullLogger = nullLoggerImpl{}

// getLogger always returns a logger.
func getLogger(cfg IfaceConfig) Logger {
	if cfg.Debug == nil {
		return nullLogger
	} else {
		return cfg.Debug
	}
}

// hexDump lazily formats binary data, matching `hexdump -C`.
//
// hexDump implements fmt.Stringer interface, allowing it to lazily dump binary
// data as hex when needed. The format of the dump matches the output of
// `hexdump -C` on the command line.
type hexDump []byte

func (h hexDump) String() string {
	var buf strings.Builder
	buf.WriteByte('\n')
	d := hex.Dumper(&buf)
	_, _ = d.Write([]byte(h))
	_ = d.Close()
	buf.WriteByte('\n')
	return buf.String()
}
