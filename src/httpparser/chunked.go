package httpparser

type OnBodyCallback func([]byte)

type chunkedBodyParser struct {
	callback       OnBodyCallback
	state          chunkedBodyState
	buffer         []byte
	chunkLength    int
	bytesReceived  int
	chunksReceived int

	maxChunkSize       int
	chunkSizeHexLength int
	chunkSizeBits      int
}

func NewChunkedBodyParser(callback OnBodyCallback, maxChunkSize int) *chunkedBodyParser {
	chunkSizeBits := getIntBits(maxChunkSize)

	return &chunkedBodyParser{
		callback:           callback,
		state:              chunkLength,
		buffer:             make([]byte, 0, maxChunkSize),
		maxChunkSize:       maxChunkSize,
		chunkSizeBits:      getIntBits(maxChunkSize),
		chunkSizeHexLength: chunkSizeBits / 4,
	}
}

func (p *chunkedBodyParser) Clear() {
	p.state = chunkLength
	p.chunkLength = 0
	p.buffer = p.buffer[:0]
	p.chunksReceived = 0
}

func (p *chunkedBodyParser) Feed(data []byte) (done bool, extraBytes []byte, err error) {
	if p.state == transferCompleted {
		/*
			It returns extra-bytes as parser must know, that it's his job now

			But if parser is feeding again, it means only that
		*/
		p.Clear()
	}
	if len(data) == 0 {
		return false, nil, nil
	}

	for i, char := range data {
		switch p.state {
		case chunkLength:
			switch char {
			case '\r':
				if p.chunkLength, err = parseHex(p.buffer); err != nil {
					p.complete()

					return true, nil, err
				}

				p.state = splitterChunkLengthCR
				p.buffer = p.buffer[:0]
			case '\n':
				if p.chunkLength, err = parseHex(p.buffer); err != nil {
					p.complete()

					return true, nil, err
				}

				p.state = chunkBody
				p.buffer = p.buffer[:0]
			default:
				// TODO: add support for trailers
				p.buffer = append(p.buffer, char)

				if len(p.buffer) > p.chunkSizeHexLength {
					p.complete()

					return true, nil, TooBigChunkSize
				}
			}
		case chunkBody:
			if p.chunkLength == 0 {
				if char == '\r' {
					p.state = splitterChunkBodyCR
					continue
				} else if char == '\n' {
					p.complete()

					return true, data[i+1:], nil
				} else {
					p.complete()

					return true, nil, InvalidChunkSplitter
				}
			}

			p.buffer = append(p.buffer, char)

			if len(p.buffer) == p.chunkLength {
				p.state = splitterChunkBodyBegin
				p.callback(p.buffer)
				p.buffer = p.buffer[:0]
			}
		case splitterChunkLengthBegin:
			switch char {
			case '\r':
				p.state = splitterChunkLengthCR
			case '\n':
				p.state = chunkBody
			default:
				p.complete()

				return true, nil, InvalidChunkSplitter
			}
		case splitterChunkLengthCR:
			if char != '\n' {
				p.complete()

				return true, nil, InvalidChunkSplitter
			}

			p.state = chunkBody
		case splitterChunkBodyBegin:
			switch char {
			case '\r':
				p.state = splitterChunkBodyCR
			case '\n':
				if p.chunkLength == 0 {
					p.complete()

					return true, data[i+1:], nil
				}

				p.state = chunkLength
			default:
				p.complete()

				return true, nil, InvalidChunkSplitter
			}
		case splitterChunkBodyCR:
			if char != '\n' {
				p.complete()

				return true, nil, InvalidChunkSplitter
			}

			if p.chunkLength == 0 {
				p.complete()

				return true, data[i+1:], nil
			}

			p.state = chunkLength
		}
	}

	return false, nil, nil
}

func (p *chunkedBodyParser) complete() {
	p.state = transferCompleted
}

// https://stackoverflow.com/questions/2274428/how-to-determine-how-many-bytes-an-integer-needs/2274457
func getIntBits(x int) int {
	if x < 0x10000 {
		if x < 0x100 {
			return 8
		} else {
			return 16
		}
	} else {
		if x < 0x100000000 {
			return 32
		} else {
			return 64
		}
	}
}
