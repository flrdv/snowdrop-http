package httpparser

import (
	"bytes"
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
	Protocol 	 		IProtocol

	splitterState 		ParsingState
	currentState 		ParsingState
	currentSplitter 	byte

	contentLength 		int64
	bodyBytesReceived 	int64
	tempBuf 			[]byte
}

func NewHTTPRequestParser(protocol IProtocol) *HTTPRequestParser {
	parser := HTTPRequestParser{Protocol: protocol}
	parser.splitterState = Nothing
	parser.currentState = Method
	parser.currentSplitter = ' '
	protocol.OnMessageBegin()

	return &parser
}

func (parser *HTTPRequestParser) Reuse(protocol *IProtocol) {
	// TODO: just re-initialize struct values
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

		if char == parser.currentSplitter {
			if char == '\n' && parser.splitterState == ReceivedLF {
				if parser.currentState != Headers {
					return true, RequestSyntaxError
				}

				parser.Protocol.OnHeadersComplete()
				parser.currentState = Body

				if index + 1 < len(data) {
					parser.parseBodyPart(data[index+1:])
				}

				break
			} else if parser.currentState == Headers {
				if err := parser.pushHeaderFromBuf(); err != nil {
					return true, err
				}
			} else {
				// warning: this potentially may cause a shit, as
				// sometimes may go po pizde as there is no default case,
				// but currently there are more priority tasks
				switch parser.currentState {
				case Method: parser.Protocol.OnMethod(parser.tempBuf)
				case Path: parser.Protocol.OnPath(parser.tempBuf)
				case Protocol:
					parser.Protocol.OnProtocol(parser.tempBuf)
					parser.Protocol.OnHeadersBegin()
				}

				parser.incState()
				parser.tempBuf = nil
			}

			if char == '\n' {
				parser.splitterState = ReceivedLF
			}
		} else {
			parser.tempBuf = append(parser.tempBuf, char)

			if parser.splitterState != Nothing {
				parser.splitterState = Nothing
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
		parser.Protocol.OnMessageComplete()

		return true
	}

	if int64(len(data)) >= parser.contentLength - parser.bodyBytesReceived {
		parser.Protocol.OnBody(data[:parser.contentLength - parser.bodyBytesReceived])
		parser.Protocol.OnMessageComplete()
		parser.currentState = MessageCompleted
		parser.bodyBytesReceived = parser.contentLength

		return true
	} else {
		parser.Protocol.OnBody(data)
		parser.bodyBytesReceived += int64(len(data))

		return false
	}
}

func (parser *HTTPRequestParser) pushHeaderFromBuf() (ok error) {
	key, value, err := parseHeader(parser.tempBuf)

	if err != nil {
		return err
	}

	parser.Protocol.OnHeader(*key, *value)
	parser.tempBuf = nil

	if strings.ToLower(*key) == "content-length" {
		contentLength, err := strconv.ParseInt(*value, 10, 0)

		if err != nil {
			return InvalidContentLengthValue
		}

		parser.contentLength = contentLength
	}

	return nil
}

func parseHeader(headersBytesString []byte) (key *string, value *string, err error) {
	for index, char := range headersBytesString {
		if char == ':' {
			key := string(headersBytesString[:index])
			value := strings.TrimPrefix(string(headersBytesString[index+1:]), " ")

			return &key, &value, nil
		}
	}

	return nil, nil, NoSplitterWasFound
}

func splitBytes(src, splitBy []byte) [][]byte {
	if len(src) == 0 {
		return [][]byte{}
	}

	var splited [][]byte
	var afterPrevSplitBy uint
	var skipIters int
	lookForward := len(splitBy)

	for index := range src[:len(src)-lookForward] {
		if skipIters > 0 {
			skipIters--
			continue
		}

		if bytes.Equal(src[index:index+lookForward], splitBy) {
			splited = append(splited, src[afterPrevSplitBy:index])
			afterPrevSplitBy = uint(index + lookForward)
			skipIters = lookForward
		}
	}

	if len(splited) == 0 {
		splited = append(splited, src)
	} else if bytes.HasSuffix(src, splitBy) {
		// if source ends with splitter, we must add pending
		// shit without counting splitter in the end
		splited = append(splited, src[afterPrevSplitBy:len(src)-lookForward])
	} else {
		// or add pending shit, but with counting everything in the end
		splited = append(splited, src[afterPrevSplitBy:])
	}

	return splited
}
