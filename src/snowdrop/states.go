package snowdrop


type (
	ParsingState uint8
	ChunkedBodyState uint8
)

const (
	MethodPathProtocol ParsingState = iota + 1
	Headers
	Body

	Dead
)

const (
	ChunkBody ChunkedBodyState = iota + 1
	ChunkLength

	/* Started splitter AFTER chunk length */
	SplitterChunkLengthBegin
	SplitterChunkLengthReceivedCR

	/* Started splitter AFTER chunk body */
	SplitterChunkBodyBegin
	SplitterChunkBodyReceivedCR

	TransferCompleted
)
