# atecc

[![pkg.go.dev][docs-badge]][docs-url]
[![Apache licensed][license-badge]][license-url]

[docs-badge]: https://pkg.go.dev/badge/github.com/northvolt/go-atecc.svg
[docs-url]: https://pkg.go.dev/github.com/northvolt/go-atecc
[license-badge]: https://img.shields.io/badge/license-Apache-blue.svg
[license-url]: https://github.com/northvolt/go-atecc/blob/main/LICENSE

Package atecc is a driver for the Microchip ATECC608 device in Go.

It supports communication using I²C and USB for dev kits.

> :warning: The API is not fully stable and may still be changed until we
> publish version 1.0.

## Availability

Cross compiling to a different platform or architecture entails disabling cgo
by default in Go. If cgo is disabled, the USB support is also disabled. For
further details, see the [usb package](https://github.com/karalabe/usb).

## Datasheets

Find all datasheets in the [Trust Platform Design Suite git
repository](https://github.com/MicrochipTech/cryptoauth_trustplatform_designsuite/).
