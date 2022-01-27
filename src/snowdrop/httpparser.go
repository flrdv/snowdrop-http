package snowdrop

import (
	"bytes"
)

const (
	DefaultBufferLength = 65535
	DefaultChunkLength  = 65535
)

var (
	cr				 = []byte("\r")
	lf 		 		 = []byte("\n")
	space 			 = []byte(" ")
	contentLength 	 = []byte("content-length")
	transferEncoding = []byte("transfer-encoding")
	chunked 		 = []byte("chunked")
)

/*
In reversed order because newer are more often than older
 */
var SupportedProtocols = [][]byte{
	[]byte("HTTP/1.1"), []byte("HTTP/1.0"), []byte("HTTP/0.9"),
}

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
	protocol 			IProtocol

	state				ParsingState
	buffer 				[]byte

	maxBufferLength 	int
	contentLength		int
	bodyBytesReceived 	int

	isChunked 			bool
 	chunksParser 		*chunkedBodyParser
}

func NewHTTPRequestParser(protocol IProtocol, maxBufferLength, maxChunkLength int) *httpRequestParser {
	if maxBufferLength < 1 {
		maxBufferLength = DefaultBufferLength
	}
	if maxChunkLength < 1 {
		maxChunkLength = DefaultChunkLength
	}

	protocol.OnMessageBegin()

	return &httpRequestParser{
		protocol: protocol,
		buffer: make([]byte, 0, maxBufferLength),
		chunksParser: NewChunkedBodyParser(protocol.OnBody, maxChunkLength),
		maxBufferLength: maxBufferLength,
		state: MethodPathProtocol,
	}
}

func (p *httpRequestParser) Clear() {
	p.state = MethodPathProtocol
	p.buffer = p.buffer[:0]
	p.contentLength = 0
	p.bodyBytesReceived = 0
	p.isChunked = false

	p.protocol.OnMessageBegin()
}

func (p *httpRequestParser) Feed(data []byte) (reqErr error) {
	/*
	This parser is absolutely stand-alone. It's like a separated sub-system in every
	server, because everything you need is just to feed it
	 */

	// TODO: make it the same as chunked parser

	if len(data) == 0 {
		return nil
	}

	switch p.state {
	case Dead: return ParserIsDead
	case Body: return p.pushBodyPiece(data)
	}

	lfCount := bytes.Count(data, lf)

	if lfCount == 0 {
		if len(p.buffer) + len(data) >= p.maxBufferLength {
			p.completeRequest()
			p.state = Dead

			return BufferSizeExceeded
		}

		p.buffer = append(p.buffer, data...)

		return nil
	}

	if len(p.buffer) > 0 {
		// after everything, we finally can append our non-empty buffer
		// and make it empty
		data = append(p.buffer, data...)
		p.buffer = p.buffer[:0]
	}

	lfIndex := bytes.IndexByte(data, '\n')

	for i := 0; i < lfCount; i++ {
		p.state, reqErr = p.parseAndGetNextState(data[:lfIndex])

		if reqErr != nil {
			return reqErr
		}

		data = data[lfIndex+1:]

		if p.state == Body {
			return p.pushBodyPiece(data)
		}

		lfIndex = bytes.IndexByte(data, '\n')
	}

	if p.state == Body {
		return p.pushBodyPiece(data)
	}

	p.buffer = data

	return nil
}

func (p *httpRequestParser) parseAndGetNextState(data []byte) (newState ParsingState, err error) {
	switch p.state {
	case MethodPathProtocol:
		method, path, protocol, err := parseMethodPathProtocolState(bytes.TrimSuffix(data, cr))

		if err != nil {
			return Dead, err
		}

		p.protocol.OnMethod(method)
		p.protocol.OnPath(path)
		p.protocol.OnProtocol(protocol)
		p.protocol.OnHeadersBegin()

		return Headers, nil
	case Headers:
		data = bytes.TrimSuffix(data, cr)

		if len(data) == 0 {
			p.protocol.OnHeadersComplete()

			if p.contentLength == 0 && !p.isChunked {
				p.completeRequest()
				p.Clear()

				return MethodPathProtocol, nil
			}

			return Body, nil
		}

		if err = p.pushHeader(data); err != nil {
			return Dead, err
		}

		return Headers, nil
	case Body:
		return Body, p.pushBodyPiece(data)
	}

	// this error must never be returned as may occur only in case
	// if current state is MessageBegin, MessageComplete, or some
	// unknown
	return Dead, AssertationError
}

func (p *httpRequestParser) completeRequest() {
	/*
	Not setting state to MethodPathProtocol, because this is a work of Clear() method,
	that is expected to be called after this method
	 */
	p.protocol.OnMessageComplete()
}

func (p *httpRequestParser) pushHeader(rawHeader []byte) error {
	key, value, err := parseHeader(rawHeader)

	if err != nil {
		return err
	}

	p.protocol.OnHeader(key, value)

	switch {
	case EqualFold(contentLength, key):
		/*
		TODO: write own implementation of ParseInt to avoid unsafe code
		  	  UPD: looks like done
		 */

		p.contentLength, err = parseUint(value)

		return err
	case EqualFold(transferEncoding, key):
		p.isChunked = EqualFold(value, chunked)

		return nil
	}

	return nil
}

func (p *httpRequestParser) pushBodyPiece(data []byte) (err error) {
	if p.isChunked {
		done, extra, err := p.chunksParser.Feed(data)

		if err != nil {
			p.state = Dead

			return err
		}

		if done {
			p.completeRequest()
			p.Clear()

			if len(extra) > 0 {
				return p.Feed(extra)
			}
		}

		return nil
	}

	if p.contentLength == 0 {
		p.completeRequest()
		p.Clear()

		return p.Feed(data)
	}

	dataLen := len(data)
	bodyBytesLeft := p.contentLength - p.bodyBytesReceived

	if bodyBytesLeft > dataLen {
		bodyBytesLeft = dataLen
	}

	p.protocol.OnBody(data[:bodyBytesLeft])
	p.bodyBytesReceived += dataLen

	if p.bodyBytesReceived >= p.contentLength {
		p.completeRequest()
		p.Clear()

		if p.bodyBytesReceived - p.contentLength > 0 {
			return p.Feed(data[bodyBytesLeft:])
		}
	}

	return nil
}

func IsProtocolSupported(proto []byte) (isSupported bool) {
	for _, supportedProto := range SupportedProtocols {
		if bytes.Equal(proto, supportedProto) {
			return true
		}
	}

	return false
}

func parseMethodPathProtocolState(data []byte) (method, path, protocol []byte, err error) {
	parsed := bytes.SplitN(data, space, 3)

	if len(parsed) != 3 {
		return nil, nil, nil, RequestSyntaxError
	}

	method = parsed[0]
	path = parsed[1]
	protocol = parsed[2]

	if !IsMethodValid(method) || len(path) == 0 || !IsProtocolSupported(protocol) {
		return nil, nil, nil, InvalidRequestData
	}

	return method, path, protocol, nil
}

func parseHeader(headersString []byte) (key, value []byte, err error) {
	for index, char := range headersString {
		if char == ':' {
			return headersString[:index], bytes.TrimPrefix(headersString[index+1:], space), nil
		}
	}

	return nil, nil, InvalidHeader
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
