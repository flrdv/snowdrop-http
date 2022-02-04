package httpparser

func isCharacterUnsafe(char byte) bool {
	if (char >= '0' && char <= '9') || (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') {
		return false
	}

	switch char {
	case '{', '}', '|', '\\', '<', '^', '>', '~', '[', ']', '"', '`', '\r', '\n', '\t', '\b':
		return true
	default:
		return false
	}
}
