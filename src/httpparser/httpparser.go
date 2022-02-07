package httpparser

import (
	ascii "github.com/scott-ainsworth/go-ascii"
)

var (
	contentLength    = []byte("content-length")
	transferEncoding = []byte("transfer-encoding")
	chunked          = []byte("chunked")
)

type IProtocol interface {
	OnMessageBegin()
	OnMethod([]byte)
	OnPath([]byte)
	OnProtocol([]byte)
	OnHeadersBegin()
	OnHeader([]byte, []byte)
	OnHeadersComplete()
	OnBody([]byte)
	OnMessageComplete()
}
type httpRequestParser struct {
	protocol IProtocol

	state             parsingState
	headerValueBegin  uint
	buffer            []byte
	reqInfoBuff       []byte
	reqInfoBuffOffset int

	maxBufferLength   int
	contentLength     int
	bodyBytesReceived int

	isChunked    bool
	chunksParser *chunkedBodyParser
}

/*
	Returns new initialized instance of parser
*/
func NewHTTPRequestParser(protocol IProtocol, settings Settings) *httpRequestParser {
	settings = PrepareSettings(settings)
	protocol.OnMessageBegin()

	return &httpRequestParser{
		protocol:        protocol,
		maxBufferLength: settings.maxBufferLength,
		buffer:          settings.Buffer,
		reqInfoBuff:     make([]byte, 0, maxMethodLength+settings.MaxPathLength+maxProtocolLength),
		chunksParser:    NewChunkedBodyParser(protocol.OnBody, settings.MaxChunkLength),
		state:           method,
	}
}

func (p *httpRequestParser) Clear() {
	p.state = method
	p.contentLength = 0
	p.bodyBytesReceived = 0
	p.isChunked = false
	p.buffer = p.buffer[:0]
	p.reqInfoBuff = p.reqInfoBuff[:0]
}

/*
	This parser is absolutely stand-alone. It's like a separated sub-system in every
	server, because everything you need is just to feed it
*/
func (p *httpRequestParser) Feed(data []byte) (reqErr error) {
	if len(data) == 0 {
		return nil
	}

	switch p.state {
	case dead:
		return ParserIsDead
	case body:
		done, extra, err := p.pushBodyPiece(data)

		if err != nil {
			p.die()

			return err
		}

		if done {
			p.Clear()
			p.protocol.OnMessageComplete()
			p.protocol.OnMessageBegin()

			if len(extra) > 0 {
				return p.Feed(extra)
			}
		}

		return nil
	}

	for i, char := range data {
		switch p.state {
		case method:
			if char == ' ' {
				if !IsMethodValid(p.reqInfoBuff) {
					p.die()

					return InvalidMethod
				}

				p.protocol.OnMethod(p.reqInfoBuff)
				p.reqInfoBuffOffset = len(p.reqInfoBuff)
				p.state = path
				continue
			}

			p.reqInfoBuff = append(p.reqInfoBuff, char)

			if len(p.reqInfoBuff) > maxMethodLength {
				p.die()

				return InvalidMethod
			}
		case path:
			if char == ' ' {
				if len(p.reqInfoBuff) == p.reqInfoBuffOffset {
					p.die()

					return InvalidPath
				}

				p.protocol.OnPath(p.reqInfoBuff[p.reqInfoBuffOffset:])
				p.reqInfoBuffOffset += len(p.reqInfoBuff[p.reqInfoBuffOffset:])
				p.state = protocol
				continue
			} else if !ascii.IsPrint(char) {
				p.die()

				return InvalidPath
			}

			p.reqInfoBuff = append(p.reqInfoBuff, char)

			if len(p.reqInfoBuff[p.reqInfoBuffOffset:]) > p.maxBufferLength {
				p.die()

				return BufferOverflow
			}
		case protocol:
			switch char {
			case '\r':
				p.state = protocolCR
			case '\n':
				p.state = protocolLF
			default:
				p.reqInfoBuff = append(p.reqInfoBuff, char)

				if len(p.reqInfoBuff[p.reqInfoBuffOffset:]) > maxProtocolLength {
					p.die()

					return BufferOverflow
				}
			}
		case protocolCR:
			if char != '\n' {
				p.die()

				return RequestSyntaxError
			}

			p.state = protocolLF
		case protocolLF:
			if !IsProtocolSupported(p.reqInfoBuff[p.reqInfoBuffOffset:]) {
				p.die()

				return ProtocolNotSupported
			}

			p.protocol.OnProtocol(p.reqInfoBuff[p.reqInfoBuffOffset:])
			p.protocol.OnHeadersBegin()

			p.buffer = append(p.buffer[:0], char)
			p.state = headerKey
		case headerKey:
			if char == ':' {
				if len(p.buffer) == 0 {
					p.die()

					return InvalidHeader
				}

				p.state = headerColon
				p.headerValueBegin = uint(len(p.buffer))
				continue
			} else if !ascii.IsPrint(char) {
				p.die()

				return InvalidHeader
			}

			p.buffer = append(p.buffer, char)

			if len(p.buffer) > p.maxBufferLength {
				p.die()

				return BufferOverflow
			}
		case headerColon:
			p.state = headerValue

			if !ascii.IsPrint(char) {
				p.die()

				return InvalidHeader
			}

			if char != ' ' {
				p.buffer = append(p.buffer, char)
			}
		case headerValue:
			switch char {
			case '\r':
				p.state = headerValueCR
			case '\n':
				p.state = headerValueLF
			default:
				if !ascii.IsPrint(char) {
					p.die()

					return InvalidHeader
				}

				p.buffer = append(p.buffer, char)

				if len(p.buffer) > p.maxBufferLength {
					p.die()

					return BufferOverflow
				}
			}
		case headerValueCR:
			if char != '\n' {
				p.die()

				return RequestSyntaxError
			}

			p.state = headerValueLF
		case headerValueLF:
			key, value := p.buffer[:p.headerValueBegin], p.buffer[p.headerValueBegin:]
			p.protocol.OnHeader(key, value)

			if EqualFold(contentLength, key) {
				var err error

				if p.contentLength, err = parseUint(value); err != nil {
					p.die()

					return InvalidContentLength
				}
			} else if EqualFold(transferEncoding, key) {
				// TODO: maybe, there are some more chunked transfers?
				p.isChunked = EqualFold(chunked, value)
			}

			switch char {
			case '\r':
				p.state = headerValueDoubleCR
			case '\n':
				p.state = body
			default:
				p.buffer = append(p.buffer[:0], char)
				p.state = headerKey
			}
		case headerValueDoubleCR:
			if char != '\n' {
				p.die()

				return RequestSyntaxError
			} else if p.contentLength == 0 && !p.isChunked {
				// TODO: save state of connection, so in case of Connection: close, I could just
				//		 receive body infinite until parser.Feed() function won't be called with
				//		 empty bytes slice

				p.Clear()
				p.protocol.OnMessageComplete()
				p.protocol.OnMessageBegin()
				continue
			}

			p.state = body
		case body:
			done, extra, err := p.pushBodyPiece(data[i:])

			if err != nil {
				p.die()

				return err
			}

			if done {
				p.Clear()
				p.protocol.OnMessageComplete()
				p.protocol.OnMessageBegin()

				if err = p.Feed(extra); err != nil {
					return err
				}
			}

			return nil
		}
	}

	return nil
}

func (p *httpRequestParser) die() {
	p.state = dead
	// anyway we don't need them anymore
	p.buffer = nil
	p.reqInfoBuff = nil
}

func (p *httpRequestParser) pushBodyPiece(data []byte) (done bool, extra []byte, err error) {
	if p.isChunked {
		done, extra, err = p.chunksParser.Feed(data)

		return done, extra, err
	}

	dataLen := len(data)
	bodyBytesLeft := p.contentLength - p.bodyBytesReceived

	if bodyBytesLeft > dataLen {
		bodyBytesLeft = dataLen
	}

	if bodyBytesLeft <= 0 {
		return true, data, nil
	}

	p.protocol.OnBody(data[:bodyBytesLeft])
	p.bodyBytesReceived += dataLen

	if p.bodyBytesReceived >= p.contentLength {
		if p.bodyBytesReceived-p.contentLength > 0 {
			return true, data[bodyBytesLeft:], nil
		}

		return true, nil, nil
	}

	return false, nil, nil
}

func IsProtocolSupported(proto []byte) (isSupported bool) {
	switch string(proto) {
	case "HTTP/1.1", "HTTP/1.0", "HTTP/0.9", // rfc recommends avoiding case-sensitive behaviour
		"http/1.1", "http/1.0", "http/0.9": // but all that strangers with Http/1.1, hTtP/1.1 are going to hell
		return true
	default:
		return false
	}
}

func EqualFold(sample, data []byte) bool {
	/*
		Works only for ascii!
	*/

	if len(sample) != len(data) {
		return false
	}

	for i, char := range sample {
		if char != (data[i] | 0x20) {
			return false
		}
	}

	return true
}
