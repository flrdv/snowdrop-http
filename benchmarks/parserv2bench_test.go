package httpparser

import (
	"testing"

	"github.com/floordiv/snowdrop/src/snowdrop"
)

type ProtocolV2 struct {}

func (p *ProtocolV2) OnMessageBegin() {}

func (p *ProtocolV2) OnMethod(_ string) {}

func (p *ProtocolV2) OnPath(_ string) {}

func (p *ProtocolV2) OnProtocol(_ string) {}

func (p *ProtocolV2) OnHeadersBegin() {}

func (p *ProtocolV2) OnHeader(_, _ string) {}

func (p *ProtocolV2) OnHeadersComplete() {}

func (p *ProtocolV2) OnBody(_ []byte) {}

func (p *ProtocolV2) OnMessageComplete() {}

var bigChromeRequest = []byte("GET / HTTP/1.1\r\nHost: localhost:8080\r\nConnection: keep-alive\r\nCache-Control: max-age=0" +
	"\r\nsec-ch-ua: \" Not A;Brand\";v=\"99\", \"Chromium\";v=\"96\", \"Google Chrome\";v=\"96\"" +
	"\r\nsec-ch-ua-mobile: ?0\r\nsec-ch-ua-platform: \"Windows\"\r\nUpgrade-Insecure-Requests: 1" +
	"\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96" +
	".0.4664.110 Safari/537.36\r\n" +
	"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8," +
	"application/signed-exchange;v=b3;q=0.9\r\nSec-Fetch-Site: none\r\nSec-Fetch-Mode: navigate" +
	"\r\nSec-Fetch-User: ?1\r\nSec-Fetch-Dest: document\r\nAccept-Encoding: gzip, deflate, br" +
	"\r\nAccept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7,uk;q=0.6\r\nCookie: csrftoken=y1y3SinAMbiYy7yn9Oc" +
	"blqbu0Ksvtdr7qrwgwgwwefdewf8JjB472GWnwfuDG; Goland-1dc491b=gegwfewfewfewf4ad0-b7ab-e4f8e1715c8b\r\n\r\n")
var smallGetRequest = []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")

func BenchmarkSmallGETRequestBy1Char(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)
	chars := divideBytes(smallGetRequest, 1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charsArr := range chars {
			parser.Feed(charsArr)
		}
	}
}

func BenchmarkSmallGETRequestBy5Chars(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)
	chars := divideBytes(smallGetRequest, 5)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charsArr := range chars {
			parser.Feed(charsArr)
		}
	}
}

func BenchmarkSmallGETRequestFull(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(smallGetRequest)
	}
}

func BenchmarkBigChromeRequestBy1Char(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)
	chars := divideBytes(bigChromeRequest, 1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charsArr := range chars {
			parser.Feed(charsArr)
		}
	}
}

func BenchmarkBigChromeRequestBy5Chars(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)
	chars := divideBytes(bigChromeRequest, 5)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, charsArr := range chars {
			parser.Feed(charsArr)
		}
	}
}

func BenchmarkBigChromeRequestFull(b *testing.B) {
	protocol := ProtocolV2{}
	parser := snowdrop.NewHTTPRequestParser(&protocol, 65535)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(bigChromeRequest)
	}
}
