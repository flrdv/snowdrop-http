package snowdrop


type (
	ParsingState uint8
	ChunkedBodyState uint8
)

const (
	MessageBegin ParsingState = iota + 1
	MethodPathProtocol
	Headers
	Body

	Dead
)

const (
	ChunkExpected ChunkedBodyState = iota + 1
	ChunkLengthExpected
	BodyCompleted
)
