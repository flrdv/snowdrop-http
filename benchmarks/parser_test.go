package httpparser

import (
	"testing"

	"github.com/floordiv/snowdrop/src/httpparser"
)

/*
This protocol is for benchmarking, that's why it's empty - to avoid
extra-overhead
*/
type Protocol struct {}

func (p *Protocol) OnMessageBegin() {}

func (p *Protocol) OnMethod(_ []byte) {}

func (p *Protocol) OnPath(_ []byte) {}

func (p *Protocol) OnProtocol(_ []byte) {}

func (p *Protocol) OnHeadersBegin() {}

func (p *Protocol) OnHeader(_, _ string) {}

func (p *Protocol) OnHeadersComplete() {}

func (p *Protocol) OnBody(_ []byte) {}

func (p *Protocol) OnMessageComplete() {}


func divideBytes(data []byte, n int) [][]byte {
	/*
	It makes from bytes array an array of bytes arrays
	 */

	var result [][]byte

	for i := 0; i < len(data); i += n {
		end := i + n

		if end > len(data) {
			end = len(data)
		}

		result = append(result, data[i:end])
	}

	return result
}

func BenchmarkSimpleGETRequestBy1CharOLDPARSER(b *testing.B) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)
	chars := divideBytes(request, 1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charArr := range chars {
			parser.Feed(charArr)
		}
	}
}
/*func BenchmarkSimpleGETRequestBy5Chars(b *testing.B) {
	request := []byte("GET / HTTP/1.1\r\nHost: rush.dev\r\nContent-Type: some content type\r\n\r\n")
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)
	chars := divideBytes(request, 5)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charArr := range chars {
			parser.Feed(charArr)
		}
	}
}*/

func BenchmarkSimpleGETRequestFullOLDPARSER(b *testing.B) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(request)
	}
}
