package httpparser

func parseUint(raw []byte) (num int, err error) {
	/*
		Tiny implementation of strconv.Atoi, but using directly bytes array,
		and returning only one error in case of shit - InvalidContentLength

		Parses 10-numeral system integers
	*/

	for _, char := range raw {
		char -= '0'

		if char > 9 {
			return 0, InvalidContentLength
		}

		num = num*10 + int(char)
	}

	return num, nil
}

func parseHex(raw []byte) (num int, err error) {
	/*
		Tiny implementation of strconv.ParseUint(raw, 16, 0), but made custom for myself,
		as I don't need all that stuff from strconv.ParseUint()

		Input data MUST be in lower-case
	*/

	for _, char := range raw {
		if (char < '0' && char > '9') && (char < 'a' && char > 'f') && (char < 'A' && char > 'F') {
			return 0, InvalidChunkSize
		}

		num = (num << 4) + int((char&0xF)+9*(char>>6))
	}

	return num, nil
}
