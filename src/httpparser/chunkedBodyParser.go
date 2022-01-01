package httpparser

import (
	"errors"
	"strconv"
)


const MaxChunkSize = 65535
const MaxHexChunkSize = "FFFF"
const ChunkSizeBitSize = 16
var TooBigChunk = errors.New("chunk overflow")
var TooBigChunkSize = errors.New("chunk size is too big")
var NotEnoughChunk = errors.New("received unexpected CRLF before the whole chunk was received")
var InvalidChunkLength = errors.New("chunk length hexdecimal is invalid")

type ChunkSizeType uint16
type OnBodyCallback func([]byte)

type chunkedBodyParser struct {
	callback 					OnBodyCallback
	state 						ParsingState
	currentChunkLength  		ChunkSizeType
	tempBuf						[]byte
	chunksReceived				int
}

func (p *chunkedBodyParser) Reuse(callback OnBodyCallback) {
	p.callback = callback
	p.state = ExpectingChunkLength
	p.currentChunkLength = 0
	p.tempBuf = nil
	p.chunksReceived = 0
}

func (p *chunkedBodyParser) Feed(data []byte) (done bool, chunkErr error) {
	if p.state == BodyCompleted {
		return true, nil
	}
	if len(data) == 0 {
		return false, nil
	}

	for _, char := range data {
		if char == '\r' {
			continue
		}

		if char == '\n' {
			if p.state == ExpectingChunkLength {
				chunkLength, err := strconv.ParseUint(string(p.tempBuf), 16, ChunkSizeBitSize)

				if err != nil {
					return true, InvalidChunkLength
				}

				p.currentChunkLength = ChunkSizeType(chunkLength)
				p.tempBuf = nil
				p.state = ExpectingChunk
			} else {
				if p.currentChunkLength == 0 {
					p.state = BodyCompleted
					return true, nil
				}
				if ChunkSizeType(len(p.tempBuf)) < p.currentChunkLength {
					return true, NotEnoughChunk
				}

				p.callback(p.tempBuf)
				p.tempBuf = nil
				p.chunksReceived++
				p.state = ExpectingChunkLength
			}
		} else {
			p.tempBuf = append(p.tempBuf, char)

			switch {
			case p.state == ExpectingChunkLength && len(p.tempBuf) > len(MaxHexChunkSize):
				return true, TooBigChunkSize
			case p.state == ExpectingChunk && len(p.tempBuf) >= MaxChunkSize:
				return true, TooBigChunk
			}
		}
	}

	return false, nil
}

func NewChunkedBodyParser(callback OnBodyCallback) *chunkedBodyParser {
	return &chunkedBodyParser{
		callback:                  callback,
		state:                     ExpectingChunkLength,
		currentChunkLength:        0,
		tempBuf:                   make([]byte, 0, MaxChunkSize),
		chunksReceived:            0,
	}
}

func quote(data []byte) string {
	return strconv.Quote(string(data))
}
