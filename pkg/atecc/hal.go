package atecc

type HAL interface {
	// Read reads up to len(p) bytes into p from the device.
	Read(p []byte) (int, error)
	// Write writes len(p) bytes from p to the device.
	Write(p []byte) (int, error)
	// idle puts the device into idle state.
	Idle() error
	// Wake wakes the device up.
	Wake() error
}
