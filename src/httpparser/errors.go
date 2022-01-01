package httpparser

import "errors"

var (
	RequestSyntaxError 			= errors.New("request syntax error")
 	InvalidHeader	 			= errors.New("invalid header line")
	TooBigChunk 				= errors.New("chunk overflow")
	TooBigChunkSize 			= errors.New("chunk size is too big")
	NotEnoughChunk 				= errors.New("received unexpected CRLF before the whole chunk was received")
	BufferSizeExceeded 			= errors.New("buffer's size is exceeded (anomaly)")

	InvalidRequestData			= errors.New("request is invalid")
	InvalidContentLengthValue 	= errors.New("invalid value for content-length header")
	InvalidChunkLength 			= errors.New("chunk length hexdecimal is invalid")
)
