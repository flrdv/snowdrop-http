package httpparser

import (
	"github.com/scott-ainsworth/go-ascii"
)

var (
	contentLength    = []byte("content-length")
	transferEncoding = []byte("transfer-encoding")
	connection       = []byte("connection")
	chunked          = []byte("chunked")
	close            = []byte("close")
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
	settings Settings

	state               parsingState
	headerValueBegin    uint
	headersBuffer       []byte
	startLineBuff       []byte
	startLineBuffOffset uint

	bodyBytesLeft int

	closeConnection bool
	isChunked       bool
	chunksParser    *chunkedBodyParser
}

/*
	Returns new initialized instance of parser
*/
func NewHTTPRequestParser(protocol IProtocol, settings Settings) *httpRequestParser {
	protocol.OnMessageBegin()
	settings = PrepareSettings(settings)

	return &httpRequestParser{
		protocol:      protocol,
		settings:      settings,
		headersBuffer: settings.HeadersBuffer,
		startLineBuff: settings.StartLineBuffer,
		chunksParser:  NewChunkedBodyParser(protocol.OnBody, settings.MaxChunkLength),
		state:         method,
	}
}

func (p *httpRequestParser) Clear() {
	p.state = method
	p.isChunked = false
	p.headersBuffer = p.headersBuffer[:0]
	p.startLineBuff = p.startLineBuff[:0]
	p.startLineBuffOffset = 0
}

/*
	This parser is absolutely stand-alone. It's like a separated sub-system in every
	server, because everything you need is just to feed it
*/
func (p *httpRequestParser) Feed(data []byte) (reqErr error) {
	if len(data) == 0 {
		if p.closeConnection {
			p.protocol.OnMessageComplete()
			p.die()

			// to let server know that we received everything, and it's time to close the connection
			return ErrConnectionClosed
		}

		return nil
	}

	switch p.state {
	case dead:
		return ErrParserIsDead
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
				if !IsMethodValid(p.startLineBuff) {
					p.die()

					return ErrInvalidMethod
				}

				p.protocol.OnMethod(p.startLineBuff)
				p.startLineBuffOffset = uint(len(p.startLineBuff))
				p.state = path
				break
			}

			p.startLineBuff = append(p.startLineBuff, char)

			if len(p.startLineBuff) > maxMethodLength {
				p.die()

				return ErrInvalidMethod
			}
		case path:
			if char == ' ' {
				if uint(len(p.startLineBuff)) == p.startLineBuffOffset {
					p.die()

					return ErrInvalidPath
				}

				p.protocol.OnPath(p.startLineBuff[p.startLineBuffOffset:])
				p.startLineBuffOffset += uint(len(p.startLineBuff[p.startLineBuffOffset:]))
				p.state = protocol
				continue
			} else if !ascii.IsPrint(char) {
				p.die()

				return ErrInvalidPath
			}

			p.startLineBuff = append(p.startLineBuff, char)

			if len(p.startLineBuff[p.startLineBuffOffset:]) > p.settings.MaxPathLength {
				p.die()

				return ErrBufferOverflow
			}
		case protocol:
			switch char {
			case '\r':
				p.state = protocolCR
			case '\n':
				p.state = protocolLF
			default:
				p.startLineBuff = append(p.startLineBuff, char)

				if len(p.startLineBuff[p.startLineBuffOffset:]) > maxProtocolLength {
					p.die()

					return ErrBufferOverflow
				}
			}
		case protocolCR:
			if char != '\n' {
				p.die()

				return ErrRequestSyntaxError
			}

			p.state = protocolLF
		case protocolLF:
			if !IsProtocolSupported(p.startLineBuff[p.startLineBuffOffset:]) {
				p.die()

				return ErrProtocolNotSupported
			}

			p.protocol.OnProtocol(p.startLineBuff[p.startLineBuffOffset:])
			p.protocol.OnHeadersBegin()

			p.headersBuffer = append(p.headersBuffer[:0], char)
			p.state = headerKey
		case headerKey:
			if char == ':' {
				if len(p.headersBuffer) == 0 {
					p.die()

					return ErrInvalidHeader
				}

				p.state = headerColon
				p.headerValueBegin = uint(len(p.headersBuffer))
				continue
			} else if !ascii.IsPrint(char) {
				p.die()

				return ErrInvalidHeader
			}

			p.headersBuffer = append(p.headersBuffer, char)

			if len(p.headersBuffer) >= p.settings.MaxHeaderLineLength {
				p.die()

				return ErrBufferOverflow
			}
		case headerColon:
			p.state = headerValue

			if !ascii.IsPrint(char) {
				p.die()

				return ErrInvalidHeader
			}

			if char != ' ' {
				p.headersBuffer = append(p.headersBuffer, char)
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

					return ErrInvalidHeader
				}

				p.headersBuffer = append(p.headersBuffer, char)

				if len(p.headersBuffer) > p.settings.MaxHeaderLineLength {
					p.die()

					return ErrBufferOverflow
				}
			}
		case headerValueCR:
			if char != '\n' {
				p.die()

				return ErrRequestSyntaxError
			}

			p.state = headerValueLF
		case headerValueLF:
			key, value := p.headersBuffer[:p.headerValueBegin], p.headersBuffer[p.headerValueBegin:]
			p.protocol.OnHeader(key, value)

			if EqualFold(contentLength, key) {
				var err error

				if p.bodyBytesLeft, err = parseUint(value); err != nil {
					p.die()

					return ErrInvalidContentLength
				}
			} else if EqualFold(transferEncoding, key) {
				// TODO: maybe, there are some more chunked transfers?
				p.isChunked = EqualFold(chunked, value)
			} else if EqualFold(connection, key) {
				p.closeConnection = EqualFold(close, value)
			}

			switch char {
			case '\r':
				p.state = headerValueDoubleCR
			case '\n':
				if p.closeConnection {
					p.state = bodyConnectionClose
					break
				}

				p.state = body
			default:
				p.headersBuffer = append(p.headersBuffer[:0], char)
				p.state = headerKey
			}
		case headerValueDoubleCR:
			if char != '\n' {
				p.die()

				return ErrRequestSyntaxError
			} else if p.closeConnection {
				p.state = bodyConnectionClose
				// anyway in case of empty bytes data it will stop parsing, so it's safe
				// but also keeps amount of body bytes limited
				p.bodyBytesLeft = p.settings.MaxBodyLength
				break
			} else if p.bodyBytesLeft == 0 && !p.isChunked {
				p.Clear()
				p.protocol.OnMessageComplete()
				p.protocol.OnMessageBegin()
				break
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
		case bodyConnectionClose:
			p.bodyBytesLeft -= len(data[i:])

			if p.bodyBytesLeft < 0 {
				p.die()

				return ErrBodyTooBig
			}

			p.protocol.OnBody(data[i:])

			return nil
		}
	}

	return nil
}

func (p *httpRequestParser) die() {
	p.state = dead
	// anyway we don't need them anymore
	p.headersBuffer = nil
	p.startLineBuff = nil
}

func (p *httpRequestParser) pushBodyPiece(data []byte) (done bool, extra []byte, err error) {
	if p.isChunked {
		done, extra, err = p.chunksParser.Feed(data)

		return done, extra, err
	}

	dataLen := len(data)

	if p.bodyBytesLeft > dataLen {
		p.protocol.OnBody(data)
		p.bodyBytesLeft -= dataLen

		return false, nil, nil
	}

	if p.bodyBytesLeft <= 0 {
		// already?? Looks like a bug
		return true, data, nil
	}

	p.protocol.OnBody(data[:p.bodyBytesLeft])

	return true, data[p.bodyBytesLeft:], nil
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
