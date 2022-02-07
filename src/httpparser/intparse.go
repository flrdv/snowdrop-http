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
			return 0, ErrInvalidContentLength
		}

		num = num*10 + int(char)
	}

	return num, nil
}
