package httpparser

import "errors"

var (
	ErrInvalidMethod        = errors.New("invalid method")
	ErrInvalidPath          = errors.New("path is empty or contains disallowed characters")
	ErrProtocolNotSupported = errors.New("protocol is not supported")
	ErrInvalidHeader        = errors.New("invalid header line")
	ErrBufferOverflow       = errors.New("buffer overflow")
	ErrInvalidContentLength = errors.New("invalid value for content-length header")
	ErrRequestSyntaxError   = errors.New("request syntax error")
	ErrBodyTooBig           = errors.New("received too much body before connection closed")

	ErrTooBigChunkSize      = errors.New("chunk size is too big")
	ErrInvalidChunkSize     = errors.New("chunk size is invalid hexdecimal value")
	ErrInvalidChunkSplitter = errors.New("invalid splitter")

	ErrConnectionClosed = errors.New("connection is closed, body has been received")
	ErrParserIsDead     = errors.New("once error occurred, parser cannot be used anymore")
)
