package httpparser

type (
	parsingState     uint8
	chunkedBodyState uint8
)

const (
	method parsingState = iota + 1
	path
	protocol
	protocolCR
	protocolLF
	headerKey
	headerColon
	headerValue
	headerValueCR
	headerValueLF
	headerValueDoubleCR
	body

	dead
)

const (
	chunkLength chunkedBodyState = iota + 1
	chunkLengthCR

	chunkBody
	chunkBodyEnd
	chunkBodyCR

	lastChunk
	lastChunkCR

	transferCompleted
)
