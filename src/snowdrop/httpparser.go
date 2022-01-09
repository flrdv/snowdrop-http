package snowdrop

import (
	"bytes"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

const DefaultBufferLength = 65535
const (
	space 			 = " "
	contentLength 	 = "content-length"
	transferEncoding = "transfer-encoding"
	chunked 		 = "chunked"
)
var (
	cr				 = []byte("\r")
	lf 		 		 = []byte("\n")
)

/*
In reversed order because newer are more often than older
 */
var SupportedProtocols = []string{
	"HTTP/1.1", "HTTP/1.0", "HTTP/0.9",
}

type IProtocol interface {
	OnMessageBegin()
	OnMethod(string)
	OnPath(string)
	OnProtocol(string)
	OnHeadersBegin()
	OnHeader(string, string)
	OnHeadersComplete()
	OnBody([]byte)
	OnMessageComplete()
}

type httpRequestParser struct {
	protocol 			IProtocol

	state				ParsingState
	buffer 				[]byte

	maxBufferLength 	int
	contentLength		uint64
	bodyBytesReceived 	uint64

	isChunked 			bool
 	chunksParser 		*chunkedBodyParser
}

func NewHTTPRequestParser(protocol IProtocol, maxBufferLength int) *httpRequestParser {
	if maxBufferLength < 1 {
		maxBufferLength = DefaultBufferLength
	}

	protocol.OnMessageBegin()

	return &httpRequestParser{
		protocol: protocol,
		buffer: make([]byte, 0, maxBufferLength),
		chunksParser: newChunkedBodyParser(protocol.OnBody),
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
	p.chunksParser.Clear()

	p.protocol.OnMessageBegin()
}

func (p *httpRequestParser) Feed(data []byte) (reqErr error) {
	/*
	This parser is absolutely stand-alone. It's like a separated sub-system in every
	server, because everything you need is just to feed it
	 */

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
		p.state, reqErr = p.parseAndGetNextState(bytes.TrimSuffix(data[:lfIndex], cr))

		if reqErr != nil {
			return reqErr
		}

		data = data[lfIndex+1:]
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
		method, path, protocol, err := parseMethodPathProtocolState(B2S(data))

		if err != nil {
			return Dead, err
		}

		p.protocol.OnMethod(method)
		p.protocol.OnPath(path)
		p.protocol.OnProtocol(protocol)
		p.protocol.OnHeadersBegin()

		return Headers, nil
	case Headers:
		if len(data) == 0 {
			p.protocol.OnHeadersComplete()

			if p.contentLength == 0 && !p.isChunked {
				p.completeRequest()
				p.Clear()

				return MethodPathProtocol, nil
			}

			return Body, nil
		}

		if err = p.pushHeader(B2S(data)); err != nil {
			return Dead, err
		}

		return Headers, nil
	case Body:
		return Body, p.pushBodyPiece(data)
	}

	// this error must never be returned as may occur only in case
	// if current state is MessageBegin, MessageComplete, or some
	// unknown
	return Dead, RequestSyntaxError
}

func (p *httpRequestParser) completeRequest() {
	/*
	Not setting state to MethodPathProtocol, because this is a work of Clear() method,
	that is expected to be called after this method
	 */
	p.protocol.OnMessageComplete()
}

func (p *httpRequestParser) pushHeader(rawHeader string) error {
	key, value, err := parseHeader(rawHeader)

	if err != nil {
		return err
	}

	p.protocol.OnHeader(key, value)

	switch {
	case strings.EqualFold(key, contentLength):
		p.contentLength, err = strconv.ParseUint(value, 10, 32)

		if err != nil {
			// why not to return the actual error? Because user expects parser's error object,
			// not from some side library (e.g. strconv)
			err = InvalidContentLengthValue
		}

		return err
	case strings.EqualFold(key, transferEncoding):
		p.isChunked = strings.EqualFold(value, chunked)

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

	dataLen := uint64(len(data))
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

func IsProtocolSupported(proto string) (isSupported bool) {
	for _, supportedProto := range SupportedProtocols {
		if proto == supportedProto {
			return true
		}
	}

	return false
}

func parseMethodPathProtocolState(data string) (method, path, protocol string, err error) {
	parsed := strings.SplitN(data, space, 3)

	if len(parsed) != 3 {
		return "", "", "", RequestSyntaxError
	}

	method = parsed[0]
	path = parsed[1]
	protocol = parsed[2]

	if !IsMethodValid(method) || len(path) == 0 || !IsProtocolSupported(protocol) {
		return "", "", "", InvalidRequestData
	}

	return method, path, protocol, nil
}

func parseHeader(headersString string) (key, value string, err error) {
	for index, char := range headersString {
		if char == ':' {
			return headersString[:index], strings.TrimPrefix(headersString[index+1:], space), nil
		}
	}

	return "", "", InvalidHeader
}

// snippet: https://github.com/valyala/fasthttp/blob/017f0aa09d7fd802bd1760836e329734ea642180/bytesconv.go#L342
// I just a bit corrected it, as I need not s2b, but b2s
//
// B2S converts bytes to a string slice without memory allocation.
//
// Note it may break if string and/or slice header will change
// in the future go versions.
func B2S(s []byte) (b string) {
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	/* #nosec G103 */
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh.Data = sh.Data
	bh.Len = sh.Len

	return b
}
