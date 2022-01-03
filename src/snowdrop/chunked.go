package snowdrop

import (
	"bytes"
	"fmt"
	"strconv"
)


const (
	_maxChunkSize = 65535
	_maxHexChunkSize = "FFFF"
)

type OnBodyCallback func([]byte)

type chunkedBodyParser struct {
	callback 					OnBodyCallback
	state 						ChunkedBodyState
	chunkLength  				uint16
	chunkBodyReceived 			int
	buffer						[]byte
	chunksReceived				int
}

func newChunkedBodyParser(callback OnBodyCallback) *chunkedBodyParser {
	return &chunkedBodyParser{
		callback:                  callback,
		state:                     ChunkLengthExpected,
		buffer:                    make([]byte, 0, _maxChunkSize),
	}
}

func (p *chunkedBodyParser) Reuse(callback OnBodyCallback) {
	p.callback = callback
	p.Clear()
}

func (p *chunkedBodyParser) Clear() {
	p.state = ChunkLengthExpected
	p.chunkLength = 0
	p.chunkBodyReceived = 0
	p.buffer = nil
	p.chunksReceived = 0
}

func (p *chunkedBodyParser) Feed(data []byte) (done bool, err error) {
	//fmt.Println("feeding with", quote(data))

	if p.state == BodyCompleted {
		return true, nil
	}
	if len(data) == 0 {
		return false, nil
	}

	lines := bytes.Split(data, lf)

	if len(p.buffer) > 0 {
		lines[0] = append(p.buffer, lines[0]...)
		p.buffer = nil
	}

	if len(lines) > 1 {
		p.buffer = lines[len(lines)-1]
		lines = lines[:len(lines)-1]
	}

	if len(lines[0]) == 0 {
		lines = lines[1:]
	}

	for _, line := range lines {
		//fmt.Println("chunk line:", quote(line))

		p.state, err = p.parseDataAndGetNextStep(bytes.TrimSuffix(line, cr))

		if p.state == BodyCompleted {
			return true, err
		}
	}

	return false, nil
}

func (p *chunkedBodyParser) parseDataAndGetNextStep(data []byte) (nextState ChunkedBodyState, err error) {
	if len(data) == 0 {
		if p.chunkLength == 0 {
			return BodyCompleted, nil
		}

		p.callback(lf)
	}

	switch p.state {
	case ChunkLengthExpected:
		if len(data) > len(_maxHexChunkSize) {
			return BodyCompleted, TooBigChunkSize
		}

		chunkLength, err := strconv.ParseUint(string(data), 16, 16)

		if err != nil {
			fmt.Println("invalid chunk size:", quote(p.buffer), quote(data))
			return BodyCompleted, InvalidChunkSize
		}

		p.chunkLength = uint16(chunkLength)

		return ChunkExpected, nil
	case ChunkExpected:
		if uint16(p.chunkBodyReceived + len(data)) > p.chunkLength {
			fmt.Println("o:", p.chunkBodyReceived, p.chunkLength, quote(data))
			return BodyCompleted, TooBigChunk
		}
		if p.chunkLength == 0 {
			return BodyCompleted, nil
		}

		p.callback(data)
		p.chunkBodyReceived += len(data)

		if uint16(p.chunkBodyReceived) == p.chunkLength {
			p.chunkBodyReceived = 0
			p.chunksReceived++

			return ChunkLengthExpected, nil
		}

		return ChunkExpected, nil
	}

	return p.state, nil
}

func quote(data []byte) string {
	/*
		Isn't removed yet as sometimes I need this for debug

		Don't ask me why I'm not using debugger. Once I used,
		an it fucked up my videodriver. I don't wanna try again
	*/

	return strconv.Quote(string(data))
}
