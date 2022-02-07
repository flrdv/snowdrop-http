package httpparser

import "math"

const (
	// hard limits
	maxMethodLength     = 7
	maxProtocolLength   = 10
	maxPathLength       = 4092           // rfc says that 65535, but it's too expensive - set it by yourself if you want
	maxHeaderLineLength = 4092           // idk what rfc says here, but this is also enough in MOST cases
	maxBodyLength       = math.MaxInt32  // 2147483647
	maxChunkLength      = math.MaxUint16 // 65535
)

const (
	// soft limits
	// to be honest, even this values for ordinary usage are unreachable
	initialPathBufferLength    = 2046
	initialHeadersBufferLength = 2046
)

type Settings struct {
	// hard limits
	MaxPathLength       int
	MaxHeaderLineLength int
	MaxBodyLength       int
	MaxChunkLength      int

	// soft limits
	InitialPathBufferLength    int
	InitialHeadersBufferLength int

	StartLineBuffer []byte
	HeadersBuffer   []byte
}

func PrepareSettings(settings Settings) Settings {
	if settings.MaxPathLength < 1 {
		settings.MaxPathLength = maxPathLength
	}
	if settings.MaxHeaderLineLength < 1 {
		settings.MaxHeaderLineLength = maxHeaderLineLength
	}
	if settings.MaxBodyLength < 1 {
		settings.MaxBodyLength = maxBodyLength
	}
	if settings.MaxChunkLength < 1 {
		settings.MaxChunkLength = maxChunkLength
	}

	if settings.InitialPathBufferLength < 1 {
		settings.InitialPathBufferLength = initialPathBufferLength
	}
	if settings.InitialHeadersBufferLength < 1 {
		settings.InitialHeadersBufferLength = initialHeadersBufferLength
	}

	if settings.StartLineBuffer == nil {
		// but user still can pass just an empty buffer with capacity he needs
		initialLength := settings.InitialPathBufferLength + maxMethodLength + maxProtocolLength
		settings.StartLineBuffer = make([]byte, 0, initialLength)
	}
	if settings.HeadersBuffer == nil {
		settings.HeadersBuffer = make([]byte, 0, settings.InitialHeadersBufferLength)
	}

	return settings
}
