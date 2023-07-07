package main

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"

	"github.com/northvolt/go-atecc/atecc"
	"github.com/peterbourgon/ff/v3/ffcli"
)

type signConfig struct {
	rootConfig *rootConfig
	in         io.Reader
	out        io.Writer
	err        io.Writer
	key        int
	signer     string
	verifier   string
}

func (c *signConfig) Exec(ctx context.Context, _ []string) error {
	var (
		signDevice   = c.signer == "device"
		verifyDevice = c.verifier == "device"
	)
	if c.rootConfig.verbose {
		fmt.Fprintf(c.err, "sign\n")
	}

	d, bus, err := newATECC(ctx, c.rootConfig)
	if err != nil {
		return err
	}
	defer bus.Close()

	if signDevice || verifyDevice {
		if locked, err := d.IsLocked(ctx, atecc.ZoneConfig); err != nil {
			return err
		} else if !locked {
			return fmt.Errorf("sign: device need to be locked before using it")
		}
	}

	var (
		priv *ecdsa.PrivateKey
		pub  crypto.PublicKey
	)
	if signDevice {
		pub, err = d.PublicKey(ctx, uint8(c.key))
		if err != nil {
			return err
		}
	} else {
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return err
		}
		pub = priv.Public()
	}

	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return err
	}
	pemPubKey := string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}))

	fmt.Fprintln(c.out, "Signing Public Key:")
	fmt.Fprintln(c.out, pemPubKey)

	h := sha256.New()
	_, err = io.Copy(h, c.in)
	if err != nil {
		return err
	}

	message := h.Sum(nil)
	fmt.Fprintln(c.out, "\nMessage Digest:")
	fmt.Fprintln(c.out, prettyHex(message))

	fmt.Fprintln(c.out, "\nSignature:")
	var signature []byte
	if signDevice {
		println("    Signing with device")
		if signature, err = d.Sign(ctx, c.key, message); err != nil {
			return err
		}
	} else {
		println("    Signing with host")
		signature, err = priv.Sign(rand.Reader, message, nil)
		if err != nil {
			return err
		}
	}

	fmt.Fprintln(c.out, prettyHex(signature))

	var verified bool
	fmt.Fprintln(c.out, "\nVerifying the signature:")
	if verifyDevice {
		fmt.Fprintln(c.out, "    Verifying with device")
		verified, err = d.VerifyExtern(ctx, message, signature, pub)
	} else {
		verified = ecdsa.VerifyASN1(pub.(*ecdsa.PublicKey), message, signature)
	}
	if err != nil {
		return err
	} else if verified {
		fmt.Fprintln(c.out, "    Signature is valid")
	} else {
		fmt.Fprintln(c.out, "    Signature is invalid")
	}

	return nil
}

func newSignCmd(
	rootConfig *rootConfig, in io.Reader, out io.Writer, err io.Writer,
) *ffcli.Command {
	cfg := signConfig{
		rootConfig: rootConfig,
		in:         in,
		out:        out,
		err:        err,
	}

	fs := flag.NewFlagSet("atecc sign", flag.ExitOnError)
	fs.IntVar(&cfg.key, "key", 0, "key id (slot number)")
	fs.StringVar(&cfg.signer, "signer", "device", "generate signature on device or host")
	fs.StringVar(&cfg.verifier, "verifier", "host", "verify signature on device or host")
	rootConfig.registerFlags(fs)

	return addLongHelp(&ffcli.Command{
		Name:       "sign",
		ShortUsage: "sign",
		ShortHelp:  "Signs and verifies the signature using the hardware.",
		FlagSet:    fs,
		Exec:       cfg.Exec,
	})
}
