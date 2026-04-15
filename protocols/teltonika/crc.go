package teltonika

// CRC16IBM computes the CRC-16/IBM (also known as CRC-16/ARC) checksum
// as specified in the Teltonika protocol documentation.
//
// Polynomial: 0xA001 (bit-reversed 0x8005)
// Initial value: 0x0000
// Input/Output reflected: yes
func CRC16IBM(data []byte) uint16 {
	var crc uint16 = 0x0000
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x0001 != 0 {
				crc = (crc >> 1) ^ 0xA001
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}
