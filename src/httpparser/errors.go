package httpparser

import "errors"

var (
	InvalidMethod        = errors.New("invalid method")
	InvalidPath          = errors.New("path is empty or contains disallowed characters")
	ProtocolNotSupported = errors.New("protocol is not supported")
	InvalidHeader        = errors.New("invalid header line")
	BufferOverflow       = errors.New("buffer overflow")
	InvalidContentLength = errors.New("invalid value for content-length header")
	RequestSyntaxError   = errors.New("request syntax error")

	TooBigChunkSize      = errors.New("chunk size is too big")
	InvalidChunkSize     = errors.New("chunk size is invalid hexdecimal value")
	InvalidChunkSplitter = errors.New("invalid splitter")

	ParserIsDead = errors.New("once error occurred, parser cannot be used anymore")
)
