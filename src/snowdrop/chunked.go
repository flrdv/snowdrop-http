package snowdrop

type OnBodyCallback func([]byte)

type chunkedBodyParser struct {
	callback 			OnBodyCallback
	state 				ChunkedBodyState
	buffer				[]byte
	chunkLength			int
	bytesReceived 		int
	chunksReceived		int

	maxChunkSize 		int
	chunkSizeHexLength	int
	chunkSizeBits		int
}

func NewChunkedBodyParser(callback OnBodyCallback, maxChunkSize int) *chunkedBodyParser {
	chunkSizeBits := getIntBits(maxChunkSize)

	return &chunkedBodyParser{
		callback: 		 	 callback,
		state:    		 	 ChunkLength,
		buffer:   		 	 make([]byte, 0, maxChunkSize),
		maxChunkSize: 	 	 maxChunkSize,
		chunkSizeBits: 		 getIntBits(maxChunkSize),
		chunkSizeHexLength:  chunkSizeBits / 4,
	}
}

func (p *chunkedBodyParser) Clear() {
	p.state = ChunkLength
	p.chunkLength = 0
	p.buffer = p.buffer[:0]
	p.chunksReceived = 0
}

func (p *chunkedBodyParser) Feed(data []byte) (done bool, extraBytes []byte, err error) {
	if p.state == TransferCompleted {
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
		// okay, let it be O(n)
		switch p.state {
		case ChunkLength:
			switch char {
			case '\r':
				if p.chunkLength, err = parseHex(p.buffer); err != nil {
					p.complete()

					return true, nil, err
				}

				p.state = SplitterChunkLengthReceivedCR
				p.buffer = p.buffer[:0]
			case '\n':
				if p.chunkLength, err = parseHex(p.buffer); err != nil {
					p.complete()

					return true, nil, err
				}

				p.state = ChunkBody
				p.buffer = p.buffer[:0]
			default:
				// TODO: add support for trailers
				p.buffer = append(p.buffer, char)

				if len(p.buffer) > p.chunkSizeHexLength {
					p.complete()

					return true, nil, TooBigChunkSize
				}
			}
		case ChunkBody:
			if p.chunkLength == 0 {
				if char == '\r' {
					p.state = SplitterChunkBodyReceivedCR
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
				p.state = SplitterChunkBodyBegin
				p.callback(p.buffer)
				p.buffer = p.buffer[:0]
			}
		case SplitterChunkLengthBegin:
			switch char {
			case '\r':
				p.state = SplitterChunkLengthReceivedCR
			case '\n':
				p.state = ChunkBody
			default:
				p.complete()

				return true, nil, InvalidChunkSplitter
			}
		case SplitterChunkLengthReceivedCR:
			if char != '\n' {
				p.complete()

				return true, nil, InvalidChunkSplitter
			}

			p.state = ChunkBody
		case SplitterChunkBodyBegin:
			switch char {
			case '\r':
				p.state = SplitterChunkBodyReceivedCR
			case '\n':
				if p.chunkLength == 0 {
					p.complete()

					return true, data[i+1:], nil
				}

				p.state = ChunkLength
			default:
				p.complete()

				return true, nil, InvalidChunkSplitter
			}
		case SplitterChunkBodyReceivedCR:
			if char != '\n' {
				p.complete()

				return true, nil, InvalidChunkSplitter
			}

			if p.chunkLength == 0 {
				p.complete()

				return true, data[i+1:], nil
			}

			p.state = ChunkLength
		}
	}

	return false, nil, nil
}

func (p *chunkedBodyParser) complete() {
	p.state = TransferCompleted
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
