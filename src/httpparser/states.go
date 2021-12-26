package httpparser


type ParsingState uint8

const (
	Method ParsingState = iota + 1
	Path
	Protocol
	Headers
	Body
	MessageCompleted
)
