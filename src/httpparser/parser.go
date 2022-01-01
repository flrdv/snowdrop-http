package httpparser

import (
	"bytes"
	"strconv"
	"strings"
)

var splittersTable = map[ParsingState]byte{
	Method: 	' ',
	Path: 		' ',
	Protocol: 	'\n',
	Headers: 	'\n',
	Body: 		'\n',
}

var SupportedProtocols = [][]byte{
	[]byte("HTTP/0.9"), []byte("HTTP/1.0"), []byte("HTTP/1.1"),
}

const MaxBufLen = 65535

type IProtocol interface {
	OnMessageBegin()
	OnMethod([]byte)
	OnPath([]byte)
	OnProtocol([]byte)
	OnHeadersBegin()
	OnHeader(string, string)
	OnHeadersComplete()
	OnBody([]byte)
	OnMessageComplete()
}

type HTTPRequestParser struct {
	protocol            IProtocol

	splitterState 		ParsingState
	currentState 		ParsingState
	currentSplitter 	byte

	contentLength 		int64
	bodyBytesReceived 	int64
	tempBuf             []byte

	isChunkedRequest	bool
	bodyChunksParser 	*chunkedBodyParser
}

func NewHTTPRequestParser(protocol IProtocol) *HTTPRequestParser {
	parser := HTTPRequestParser{
		protocol: 			protocol,
		splitterState:  	Nothing,
		currentState: 		Method,
		currentSplitter: 	splittersTable[Method],
		tempBuf: 			make([]byte, 0, MaxBufLen),
		isChunkedRequest: 	false,
		bodyChunksParser: 	NewChunkedBodyParser(protocol.OnBody),
	}
	protocol.OnMessageBegin()

	return &parser
}

func (parser *HTTPRequestParser) Reuse(protocol IProtocol) {
	parser.protocol = protocol
	parser.splitterState = Nothing
	parser.currentState = Method
	parser.currentSplitter = splittersTable[parser.currentState]
	parser.contentLength = 0
	parser.bodyBytesReceived = 0

	if parser.isChunkedRequest {
		parser.isChunkedRequest = false
		parser.bodyChunksParser.Reuse(protocol.OnBody)
	}
	// tempBuf must be already empty as it is empty while headers are completely parsed
	protocol.OnMessageBegin()
}

func (parser *HTTPRequestParser) GetState() ParsingState {
	return parser.currentState
}

func (parser *HTTPRequestParser) Feed(data []byte) (completed bool, requestError error) {
	if parser.currentState == MessageCompleted {
		return true, nil
	} else if parser.currentState == Body {
		if parser.isChunkedRequest {
			done, err := parser.bodyChunksParser.Feed(data)

			if err != nil {
				parser.completeMessageNoCallback()

				return true, err
			}

			if done {
				parser.completeMessage()
			}

			return done, nil
		} else {
			done := parser.parseBodyPart(data)

			return done, nil
		}
	}

	for index, char := range data {
		if char == '\r' {
			// possibly security issue, but the most convenient
			// solution for parsing http
			continue
		}

		if char != parser.currentSplitter {
			parser.tempBuf = append(parser.tempBuf, char)

			if parser.splitterState != Nothing {
				parser.splitterState = Nothing
			}
		} else {
			if char == '\n' && parser.splitterState == ReceivedLF {
				if parser.currentState != Headers {
					parser.completeMessageNoCallback()

					return true, RequestSyntaxError
				}

				parser.protocol.OnHeadersComplete()
				parser.currentState = Body

				if index + 1 < len(data) && !parser.isChunkedRequest {
					isMessageCompleted := parser.parseBodyPart(data[index+1:])

					if isMessageCompleted {
						parser.completeMessage()
					}

					return isMessageCompleted, nil
				} else if parser.isChunkedRequest {
					done, err := parser.bodyChunksParser.Feed(data[index+1:])


					if err != nil {
						parser.completeMessageNoCallback()

						return true, err
					}
					if done {
						parser.completeMessage()
					}

					return done, nil
				} else if parser.contentLength == 0 {
					parser.completeMessage()

					return true, nil
				}

				return false, nil
			} else if parser.currentState == Headers {
				if err := parser.pushHeaderFromBuf(); err != nil {
					parser.completeMessageNoCallback()

					return true, err
				}
			} else {
				// warning: this potentially may cause a shit, as
				// sometimes may go po pizde as there is no default case,
				// but currently there are more priority tasks
				switch parser.currentState {
				case Method:
					if !IsMethodValid(parser.tempBuf) {
						parser.completeMessageNoCallback()

						return true, InvalidRequestData
					}

					parser.protocol.OnMethod(parser.tempBuf)
				case Path:
					if len(parser.tempBuf) == 0 {
						parser.completeMessageNoCallback()

						return true, InvalidRequestData
					}

					parser.protocol.OnPath(parser.tempBuf)
				case Protocol:
					if !isProtocolValid(parser.tempBuf) {
						parser.completeMessageNoCallback()

						return true, InvalidRequestData
					}

					parser.protocol.OnProtocol(parser.tempBuf)
					parser.protocol.OnHeadersBegin()
				}

				parser.incState()
				parser.tempBuf = nil
			}

			if char == '\n' {
				parser.splitterState = ReceivedLF
			}
		}
	}

	return false, nil
}

func (parser *HTTPRequestParser) incState() {
	parser.currentState++  // dirty trick, but we're not about clean code here
	parser.currentSplitter = splittersTable[parser.currentState]
}

func (parser *HTTPRequestParser) parseBodyPart(data []byte) (completed bool) {
	if parser.contentLength == 0 {
		parser.completeMessage()

		return true
	}

	if int64(len(data)) >= parser.contentLength - parser.bodyBytesReceived {
		parser.protocol.OnBody(data[:parser.contentLength - parser.bodyBytesReceived])
		parser.completeMessage()
		parser.bodyBytesReceived = parser.contentLength

		return true
	} else {
		parser.protocol.OnBody(data)
		parser.bodyBytesReceived += int64(len(data))

		return false
	}
}

func (parser *HTTPRequestParser) completeMessage() {
	parser.currentState = MessageCompleted
	parser.protocol.OnMessageComplete()
}

func (parser *HTTPRequestParser) completeMessageNoCallback() {
	/*
	The difference is that this method used usually after errors
	occurred, so message is completed and can't be parsed anymore
	 */
	parser.currentState = MessageCompleted
}

func (parser *HTTPRequestParser) pushHeaderFromBuf() (err error) {
	key, value, err := parseHeader(parser.tempBuf)

	if err != nil {
		return err
	}

	parser.protocol.OnHeader(key, value)
	parser.tempBuf = nil

	switch strings.ToLower(key) {
	case "content-length":
		contentLength, err := strconv.ParseInt(value, 10, 0)

		if err != nil {
			return InvalidContentLengthValue
		}

		parser.contentLength = contentLength
	case "transfer-encoding":
		parser.isChunkedRequest = strings.ToLower(value) == "chunked"
	}

	return nil
}

func isProtocolValid(proto []byte) bool {
	for _, supportedProto := range SupportedProtocols {
		if bytes.Equal(supportedProto, proto) {
			return true
		}
	}

	return false
}

func parseHeader(headersBytesString []byte) (key, value string, err error) {
	for index, char := range headersBytesString {
		if char == ':' {
			key := string(headersBytesString[:index])
			value := strings.TrimPrefix(string(headersBytesString[index+1:]), " ")

			return key, value, nil
		}
	}

	return "", "", NoSplitterWasFound
}
