package snowdrop

import (
	"bytes"
	"fmt"
	"strconv"
)

const DefaultBufferLength = 65535
var (
	space 			 = []byte(" ")
	cr				 = []byte("\r")
	lf 		 		 = []byte("\n")
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
	contentLength		uint64
	bodyBytesReceived 	uint64

	isChunked 			bool
 	chunksParser 		*chunkedBodyParser
}

func NewHTTPRequestParser(protocol IProtocol, maxBufferLength int) *httpRequestParser {
	if maxBufferLength < 1 {
		maxBufferLength = DefaultBufferLength
	}

	return &httpRequestParser{
		protocol: protocol,
		buffer: make([]byte, 0, maxBufferLength),
		chunksParser: newChunkedBodyParser(protocol.OnBody),
		maxBufferLength: maxBufferLength,
		state: MessageBegin,
	}
}

func (p *httpRequestParser) Clear() {
	p.state = MethodPathProtocol
	p.buffer = nil
	p.contentLength = 0
	p.bodyBytesReceived = 0
	p.isChunked = false
	p.chunksParser.Clear()

	p.protocol.OnMessageBegin()
}

func (p *httpRequestParser) Feed(data []byte) (requestCompleted bool, extraBytes []byte, requestError error) {
	if p.state == MessageCompleted {
		p.Clear()
	}

	if len(data) == 0 {
		return false, nil, nil
	}

	switch p.state {
	case MessageBegin:
		p.protocol.OnMessageBegin()
		p.state = MethodPathProtocol
	case Body:
		if len(p.buffer) > 0 {
			data = append(p.buffer, data...)
			p.buffer = nil
		}

		requestCompleted, extraBytes, requestError = p.pushBodyPeace(data)

		if requestCompleted {
			// OnMessageComplete is called only in case of correct request
			// if any error occurred during parsing - it won't be called
			p.completeRequest(requestError == nil)


			return true, extraBytes, requestError
		} else if requestError != nil {
			p.completeRequest(false)

			return true, nil, requestError
		}

		return false, nil, requestError
	case MessageCompleted:
		p.Clear()
	}

	foodLines := bytes.Split(data, lf)

	if len(foodLines) == 1 {
		nonCompletedLine := foodLines[0]

		if len(p.buffer)+len(nonCompletedLine) >= p.maxBufferLength {
			p.buffer = nil
			p.completeRequest(false)

			return true, nil, BufferSizeExceeded
		}

		p.buffer = append(p.buffer, nonCompletedLine...)

		return false, nil, nil
	}

	if len(p.buffer) > 0 {
		// if we finally have some LF, but also something in
		// buffer, we need to push this something from buffer
		// to processing data
		foodLines[0] = append(p.buffer, foodLines[0]...)
		p.buffer = nil
	}

	lastFoodPeaceLen := len(foodLines[len(foodLines)-1])

	if lastFoodPeaceLen > 0 {
		// at this point buffer is always empty
		p.buffer = foodLines[len(foodLines)-1]
		foodLines = foodLines[:len(foodLines)-1]
	} else if lastFoodPeaceLen == 0 {
		foodLines = foodLines[:len(foodLines)-1]
	}

	for index, line := range foodLines {
		p.state, extraBytes, requestError = p.parseAndGetNextState(bytes.TrimSuffix(line, cr))

		if p.state == MessageCompleted {
			p.completeRequest(requestError == nil)

			if index + 1 < len(foodLines) {
				extraBytes = collapseByteArrays(extraBytes, foodLines[index+1:]...)
			}

			return true, extraBytes, requestError
		} else if p.state == Body && p.isChunked {
			// as chunks has a bit different logic, we have to rely on
			// chunked body parser, as otherwise we may trap in a shit
			// like buffer overflow (or near to overflow, smbd may raise
			// a memory leak keeping connection opened & keeping 65534
			// bytes sent, for example)

			foodLen := len(foodLines)

			if index + 1 < foodLen {
				dataPiece := bytes.TrimSuffix(foodLines[index], cr)

				for _, arr := range foodLines[index+1:] {
					dataPiece = append(dataPiece, append(arr, lf...)...)
				}

				//fmt.Println("wanna feed:", quote(append(dataPiece, p.buffer...)))
				requestCompleted, requestError = p.chunksParser.Feed(append(dataPiece, p.buffer...))

				if requestCompleted {
					p.completeRequest(requestError == nil)
				}

				p.buffer = nil

				return requestCompleted, nil, requestError
			}

			return p.state == MessageCompleted, nil, nil
		}
	}

	if p.state == Body && len(p.buffer) > 0 {
		requestCompleted, extraBytes, requestError = p.pushBodyPeace(p.buffer)
		p.buffer = nil

		if requestCompleted {
			p.completeRequest(requestError == nil)
		}

		return requestCompleted, extraBytes, requestError
	}

	return false, nil, nil
}

func (p *httpRequestParser) parseAndGetNextState(data []byte) (newState ParsingState, extraBytes []byte, err error) {
	switch p.state {
	case MethodPathProtocol:
		method, path, protocol, err := parseMethodPathProtocolState(data)

		if err != nil {
			return MessageCompleted, nil, err
		}

		p.protocol.OnMethod(method)
		p.protocol.OnPath(path)
		p.protocol.OnProtocol(protocol)
		p.protocol.OnHeadersBegin()

		return Headers, extraBytes, nil
	case Headers:
		if len(data) == 0 {
			p.protocol.OnHeadersComplete()

			if p.contentLength == 0 && !p.isChunked {
				return MessageCompleted, nil, nil
			}

			return Body, nil, nil
		}

		if err := p.pushHeader(data); err != nil {
			return MessageCompleted, nil, err
		}

		return Headers, nil, nil
	case Body:
		requestCompleted, extra, err := p.pushBodyPeace(data)

		if requestCompleted {
			return MessageCompleted, extra, err
		}

		return Body, nil, err
	}

	// this error must never be returned as may occur only in case
	// if current state is MessageBegin, MessageComplete, or some
	// unknown
	return MessageCompleted, nil, RequestSyntaxError
}

func (p *httpRequestParser) completeRequest(callback bool) {
	if callback {
		p.protocol.OnMessageComplete()
	}

	p.state = MessageCompleted
}

func (p *httpRequestParser) pushHeader(rawHeader []byte) error {
	key, value, err := parseHeader(rawHeader)
	key = bytes.ToLower(key)

	if err != nil {
		return err
	}

	p.protocol.OnHeader(key, value)

	switch {
	case bytes.Equal(key, contentLength):
		p.contentLength, err = strconv.ParseUint(string(value), 10, 32)

		if err != nil {
			err = InvalidContentLengthValue
		}

		return err
	case bytes.Equal(key, transferEncoding):
		p.isChunked = bytes.Equal(bytes.ToLower(value), chunked)

		return nil
	}

	return nil
}

func (p *httpRequestParser) pushBodyPeace(peace []byte) (reqCompleted bool, extra []byte, err error) {
	if p.isChunked {
		// TODO: make this also return extra-bytes
		done, err := p.chunksParser.Feed(peace)

		return done, nil, err
	}

	peaceLen := uint64(len(peace))
	totalBodyBytes := p.bodyBytesReceived + peaceLen

	if totalBodyBytes > p.contentLength {
		extraBytes := totalBodyBytes - p.contentLength
		extra = peace[extraBytes+1:]
		peace = peace[:extraBytes+1]
		peaceLen -= extraBytes
		reqCompleted = true
	}

	p.bodyBytesReceived += peaceLen
	p.protocol.OnBody(peace)

	return p.bodyBytesReceived == p.contentLength, extra, nil
}

func IsProtocolSupported(proto []byte) (isSupported bool) {
	for _, supportedProto := range SupportedProtocols {
		if bytes.Equal(supportedProto, proto) {
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

func parseHeader(headersBytesString []byte) (key, value []byte, err error) {
	for index, char := range headersBytesString {
		if char == ':' {
			return headersBytesString[:index], bytes.TrimPrefix(headersBytesString[index+1:], space), nil
		}
	}

	return nil, nil, InvalidHeader
}

func collapseByteArrays(base []byte, arrays ...[]byte) []byte {
	if len(arrays) == 0 {
		return base
	}

	for _, arr := range arrays[1:] {
		base = append(base, arr...)
	}

	return base
}

func printArrs(data [][]byte) {
	for _, arr := range data {
		fmt.Print(quote(arr), " ")
	}

	fmt.Println()
}
