package atecc

// crc16 calculates the CRC.
//
// Refer to the Atmel CryptoAuthentication Data Zone CRC Calculation document
// for details about how CRC is used in this device.
// https://ww1.microchip.com/downloads/en/Appnotes/Atmel-8936-CryptoAuth-Data-Zone-CRC-Calculation-ApplicationNote.pdf
func crc16(data []byte) uint16 {
	var polynom uint16 = 0x8005
	var crc uint16

	for _, b := range data {
		for j := 0; j < 8; j++ {
			var data_bit byte
			if b&(1<<j) != 0 {
				data_bit = 1
			}
			crc_bit := byte(crc >> 15)
			crc = crc << 1
			if data_bit != crc_bit {
				crc = crc ^ polynom
			}
		}
	}

	return crc
}
