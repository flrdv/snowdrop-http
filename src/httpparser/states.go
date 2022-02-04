package httpparser

type (
	parsingState     uint8
	chunkedBodyState uint8
	//ParserState      uint8
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
	chunkBody chunkedBodyState = iota + 1
	chunkLength

	/* Started splitter AFTER chunk length */
	splitterChunkLengthBegin
	splitterChunkLengthCR

	/* Started splitter AFTER chunk body */
	splitterChunkBodyBegin
	splitterChunkBodyCR

	transferCompleted
)
