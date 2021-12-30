package httpparser

import (
	"errors"
	"strconv"
	"strings"
)


var RequestSyntaxError = errors.New("request syntax error")
var InvalidContentLengthValue = errors.New("invalid value for content-length header")
var NoSplitterWasFound = errors.New("no splitter was found")

var splittersTable = map[ParsingState]byte{
	Method: ' ',
	Path: ' ',
	Protocol: '\n',
	Headers: '\n',
	Body: '\n',
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
	protocol 	 		IProtocol

	splitterState 		ParsingState
	currentState 		ParsingState
	currentSplitter 	byte

	contentLength 		int64
	bodyBytesReceived 	int64
	tempBuf 			[]byte
}

func NewHTTPRequestParser(protocol IProtocol) *HTTPRequestParser {
	parser := HTTPRequestParser{
		protocol: protocol,
		splitterState: Nothing,
		currentState: Method,
		currentSplitter: ' ',
		tempBuf: make([]byte, 0, MaxBufLen),
	}
	protocol.OnMessageBegin()

	return &parser
}

func (parser *HTTPRequestParser) Reuse(protocol IProtocol) {
	parser.protocol = protocol
	parser.splitterState = Nothing
	parser.currentState = Method
	parser.currentSplitter = ' '
	parser.contentLength = 0
	parser.bodyBytesReceived = 0
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
		done := parser.parseBodyPart(data)

		return done, nil
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
					return true, RequestSyntaxError
				}

				parser.protocol.OnHeadersComplete()
				parser.currentState = Body

				if index + 1 < len(data) {
					done := parser.parseBodyPart(data[index+1:])

					return done, nil
				}

				return false, nil
			} else if parser.currentState == Headers {
				if err := parser.pushHeaderFromBuf(); err != nil {
					return true, err
				}
			} else {
				// warning: this potentially may cause a shit, as
				// sometimes may go po pizde as there is no default case,
				// but currently there are more priority tasks
				switch parser.currentState {
				case Method: parser.protocol.OnMethod(parser.tempBuf)
				case Path: parser.protocol.OnPath(parser.tempBuf)
				case Protocol:
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
		parser.currentState = MessageCompleted
		parser.protocol.OnMessageComplete()

		return true
	}

	if int64(len(data)) >= parser.contentLength - parser.bodyBytesReceived {
		parser.protocol.OnBody(data[:parser.contentLength - parser.bodyBytesReceived])
		parser.protocol.OnMessageComplete()
		parser.currentState = MessageCompleted
		parser.bodyBytesReceived = parser.contentLength

		return true
	} else {
		parser.protocol.OnBody(data)
		parser.bodyBytesReceived += int64(len(data))

		return false
	}
}

func (parser *HTTPRequestParser) pushHeaderFromBuf() (ok error) {
	key, value, err := parseHeader(parser.tempBuf)

	if err != nil {
		return err
	}

	parser.protocol.OnHeader(key, value)
	parser.tempBuf = nil

	if strings.ToLower(key) == "content-length" {
		contentLength, err := strconv.ParseInt(value, 10, 0)

		if err != nil {
			return InvalidContentLengthValue
		}

		parser.contentLength = contentLength
	}

	return nil
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
