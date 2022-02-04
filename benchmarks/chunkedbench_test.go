package httpparser

import (
	"bytes"
	"fmt"
	"github.com/floordiv/snowdrop/src/httpparser"
	"testing"
)

func BenchmarkSmallChunkedBody(b *testing.B) {
	parser := httpparser.NewChunkedBodyParser(func(_ []byte) {}, 65535)
	sampleChunkedBody := []byte("d\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\nok")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(sampleChunkedBody)
	}
}

func Benchmark100BigChunks(b *testing.B) {
	parser := httpparser.NewChunkedBodyParser(func(_ []byte) {}, 65535)
	var data []byte

	for i := 0; i < 100; i++ {
		data = append(data, 'F', 'F', 'F', 'E', '\r', '\n')
		data = append(data, bytes.Repeat([]byte("a"), 65534)...)
		data = append(data, '\r', '\n')
	}

	data = append(data, '0', '\r', '\n', '\r', '\n')

	fmt.Println(len(data))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(data)
	}
}
