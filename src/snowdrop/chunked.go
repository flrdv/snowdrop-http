package snowdrop

import (
	"bytes"
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
	chunkLength  				int
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
	p.buffer = p.buffer[:0]
	p.chunksReceived = 0
}

func (p *chunkedBodyParser) Feed(data []byte) (done bool, extraBytes []byte, err error) {
	if p.state == BodyCompleted {
		// it's not http parser, here we don't care about streaming parse
		return true, nil, nil
	}
	if len(data) == 0 || bytes.Equal(data, cr) {
		return false, nil, nil
	}

	lfCount := bytes.Count(data, lf)

	if lfCount == 0 {
		done, err = p.parseStateDataPiece(bytes.TrimSuffix(data, cr))

		return done, nil, err
	} else {
		nextLF := bytes.IndexByte(data, '\n')

		for i := 0; i < lfCount; i++ {
			piece := bytes.TrimPrefix(data[:nextLF], cr)
			done, err = p.parseStateData(piece)

			if err != nil {
				return true, nil, err
			}
			if done {
				return true, data[nextLF+1:], nil
			}

			data = data[nextLF+1:]
			nextLF = bytes.IndexByte(data, '\n')
		}

		// this must be already the last one sliced element
		done, err = p.parseStateDataPiece(data)

		return done, nil, err
	}
}

func (p *chunkedBodyParser) parseStateDataPiece(data []byte) (done bool, err error) {
	switch p.state {
	case ChunkExpected:
		if p.chunkBodyReceived + len(data) > p.chunkLength {
			p.state = BodyCompleted

			return true, TooBigChunk
		} else if p.chunkLength == 0 {
			return true, nil
		}

		if p.pushBody(data) {
			return true, nil
		}
	case ChunkLengthExpected:
		if len(p.buffer)  > len(_maxHexChunkSize) {
			p.state = BodyCompleted

			return true, TooBigChunkSize
		}

		p.buffer = append(p.buffer, data...)

		return false, nil
	}

	// this also shouldn't ever happen
	// i hope
	p.state = BodyCompleted

	return true, AssertationError
}

func (p *chunkedBodyParser) parseStateData(data []byte) (done bool, err error) {
	switch p.state {
	case ChunkExpected:
		if p.chunkBodyReceived + len(data) > p.chunkLength {
			// in this handler we're passing only whole pieces, so yep
			return true, TooBigChunk
		}

		done = p.pushBody(data)

		if !done {
			p.chunkBodyReceived += len(data)
		}

		return false, nil
	case ChunkLengthExpected:
		rawChunkLength := B2S(p.buffer)
		chunkLength, err := strconv.ParseInt(rawChunkLength, 16, 32)

		if err != nil {
			p.state = BodyCompleted

			return true, InvalidChunkSize
		}

		p.chunkLength = int(chunkLength)

		return false, nil
	}

	p.state = BodyCompleted

	// this shouldn't ever happen
	// i hope
	return true, AssertationError
}

func (p *chunkedBodyParser) pushBody(data []byte) (done bool) {
	p.callback(data)
	p.chunkBodyReceived += len(data)

	if p.chunkBodyReceived == p.chunkLength {
		p.chunkBodyReceived = 0
		p.chunksReceived++
		p.state = ChunkLengthExpected

		return true
	}

	return false
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

		chunkLength, err := strconv.ParseInt(string(data), 16, 32)

		if err != nil {
			return BodyCompleted, InvalidChunkSize
		} else if chunkLength > _maxChunkSize {
			return BodyCompleted, TooBigChunkSize
		}

		p.chunkLength = int(chunkLength)

		return ChunkExpected, nil
	case ChunkExpected:
		if p.chunkBodyReceived + len(data) > p.chunkLength {
			return BodyCompleted, TooBigChunk
		}
		if p.chunkLength == 0 {
			return BodyCompleted, nil
		}

		p.callback(data)
		p.chunkBodyReceived += len(data)

		if p.chunkBodyReceived == p.chunkLength {
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
