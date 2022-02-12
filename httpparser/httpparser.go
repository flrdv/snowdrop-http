package httpparser

import (
	"github.com/scott-ainsworth/go-ascii"
)

var (
	contentLength    = []byte("content-length")
	transferEncoding = []byte("transfer-encoding")
	connection       = []byte("connection")
	chunked          = []byte("chunked")
	closeConnection  = []byte("close")
)

type IProtocol interface {
	OnMessageBegin() error
	OnMethod([]byte) error
	OnPath([]byte) error
	OnProtocol([]byte) error
	OnHeadersBegin() error
	OnHeader([]byte, []byte) error
	OnHeadersComplete() error
	OnBody([]byte) error
	OnMessageComplete() error
}
type httpRequestParser struct {
	protocol IProtocol
	settings Settings

	state            parsingState
	headerValueBegin uint
	headersBuffer    []byte
	startLineBuff    []byte
	startLineOffset  uint

	bodyBytesLeft int

	closeConnection bool
	isChunked       bool
	chunksParser    *chunkedBodyParser
}

/*
	Returns new initialized instance of parser
*/
func NewHTTPRequestParser(protocol IProtocol, settings Settings) (*httpRequestParser, error) {
	if err := protocol.OnMessageBegin(); err != nil {
		return nil, err
	}

	settings = PrepareSettings(settings)

	return &httpRequestParser{
		protocol:      protocol,
		settings:      settings,
		headersBuffer: settings.HeadersBuffer,
		startLineBuff: settings.StartLineBuffer,
		chunksParser:  NewChunkedBodyParser(protocol.OnBody, settings.MaxChunkLength),
		state:         method,
	}, nil
}

func (p *httpRequestParser) Clear() {
	p.state = method
	p.isChunked = false
	p.headersBuffer = p.headersBuffer[:0]
	p.startLineBuff = p.startLineBuff[:0]
	p.startLineOffset = 0
}

/*
	This parser is absolutely stand-alone. It's like a separated sub-system in every
	server, because everything you need is just to feed it
*/
func (p *httpRequestParser) Feed(data []byte) (reqErr error) {
	if len(data) == 0 {
		if p.closeConnection {
			p.die()

			if reqErr = p.protocol.OnMessageComplete(); reqErr != nil {
				return reqErr
			}

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

			if reqErr = p.protocol.OnMessageComplete(); reqErr != nil {
				p.die()

				return err
			}

			if reqErr = p.protocol.OnMessageBegin(); reqErr != nil {
				p.die()

				return reqErr
			}

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

				if reqErr = p.protocol.OnMethod(p.startLineBuff); reqErr != nil {
					p.die()

					return reqErr
				}

				p.startLineOffset = uint(len(p.startLineBuff))
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
				if uint(len(p.startLineBuff)) == p.startLineOffset {
					p.die()

					return ErrInvalidPath
				}

				if reqErr = p.protocol.OnPath(p.startLineBuff[p.startLineOffset:]); reqErr != nil {
					p.die()

					return reqErr
				}

				p.startLineOffset += uint(len(p.startLineBuff[p.startLineOffset:]))
				p.state = protocol
				continue
			} else if !ascii.IsPrint(char) {
				p.die()

				return ErrInvalidPath
			}

			p.startLineBuff = append(p.startLineBuff, char)

			if len(p.startLineBuff[p.startLineOffset:]) > p.settings.MaxPathLength {
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

				if len(p.startLineBuff[p.startLineOffset:]) > maxProtocolLength {
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
			if !IsProtocolSupported(p.startLineBuff[p.startLineOffset:]) {
				p.die()

				return ErrProtocolNotSupported
			}

			if reqErr = p.protocol.OnProtocol(p.startLineBuff[p.startLineOffset:]); reqErr != nil {
				p.die()

				return reqErr
			}
			if reqErr = p.protocol.OnHeadersBegin(); reqErr != nil {
				p.die()

				return reqErr
			}

			if char == '\r' {
				p.state = headerValueDoubleCR
				break
			} else if char == '\n' {
				if reqErr = p.protocol.OnHeadersComplete(); reqErr != nil {
					p.die()

					return reqErr
				}
				if reqErr = p.protocol.OnMessageComplete(); reqErr != nil {
					p.die()

					return reqErr
				}

				p.Clear()
				break
			} else if !ascii.IsPrint(char) || char == ':' {
				p.die()

				return ErrInvalidHeader
			}

			p.headersBuffer = append(p.headersBuffer, char)
			p.state = headerKey
		case headerKey:
			if char == ':' {
				p.state = headerColon
				p.headerValueBegin = uint(len(p.headersBuffer))
				break
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

			if reqErr = p.protocol.OnHeader(key, value); reqErr != nil {
				p.die()

				return reqErr
			}

			switch len(key) {
			case len(contentLength):
				good := true

				for j, character := range contentLength {
					if character != (key[j] | 0x20) {
						good = false
						break
					}
				}

				if good {
					var err error

					if p.bodyBytesLeft, err = parseUint(value); err != nil {
						p.die()

						return ErrInvalidContentLength
					}
				}
			case len(transferEncoding):
				good := true

				for j, character := range transferEncoding {
					if character != (key[j] | 0x20) {
						good = false
						break
					}
				}

				if good {
					// TODO: maybe, there are some more transfer encodings I must support?
					p.isChunked = EqualFold(chunked, value)
				}
			case len(connection):
				good := true

				for j, character := range connection {
					if character != (key[j] | 0x20) {
						good = false
						break
					}
				}

				if good {
					p.closeConnection = EqualFold(closeConnection, value)
				}
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

				if reqErr = p.protocol.OnMessageComplete(); reqErr != nil {
					p.die()

					return reqErr
				}
				if reqErr = p.protocol.OnMessageBegin(); reqErr != nil {
					p.die()

					return reqErr
				}

				p.Clear()
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

				if reqErr = p.protocol.OnMessageComplete(); reqErr != nil {
					p.die()

					return reqErr
				}
				if reqErr = p.protocol.OnMessageBegin(); reqErr != nil {
					p.die()

					return reqErr
				}

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

			if reqErr = p.protocol.OnBody(data[i:]); reqErr != nil {
				p.die()

				return reqErr
			}

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
		if err = p.protocol.OnBody(data); err != nil {
			return true, nil, err
		}

		p.bodyBytesLeft -= dataLen

		return false, nil, nil
	}

	if p.bodyBytesLeft <= 0 {
		// already?? Looks like a bug
		return true, data, nil
	}

	if err = p.protocol.OnBody(data[:p.bodyBytesLeft]); err != nil {
		return true, nil, err
	}

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
