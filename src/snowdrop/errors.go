package snowdrop

import "errors"

var (
	RequestSyntaxError 			= errors.New("request syntax error")
	InvalidHeader	 			= errors.New("invalid header line")
	BufferSizeExceeded 			= errors.New("buffer's size is exceeded (anomaly)")
	InvalidRequestData			= errors.New("request is invalid")
	InvalidContentLength    	= errors.New("invalid value for content-length header")

	TooBigChunkSize 			= errors.New("chunk size is too big")
	InvalidChunkSize			= errors.New("chunk size is invalid hexdecimal value")
	InvalidChunkSplitter		= errors.New("invalid splitter")

	AssertationError			= errors.New("BUG")
	ParserIsDead 				= errors.New("once error occurred, parser cannot be used anymore")
)
