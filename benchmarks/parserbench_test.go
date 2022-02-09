package httpparser

import (
	"testing"

	"github.com/floordiv/snowdrop/httpparser"
)

type ProtocolV2 struct{}

func (p ProtocolV2) OnMessageBegin() error      { return nil }
func (p ProtocolV2) OnMethod(_ []byte) error    { return nil }
func (p ProtocolV2) OnPath(_ []byte) error      { return nil }
func (p ProtocolV2) OnProtocol(_ []byte) error  { return nil }
func (p ProtocolV2) OnHeadersBegin() error      { return nil }
func (p ProtocolV2) OnHeader(_, _ []byte) error { return nil }
func (p ProtocolV2) OnHeadersComplete() error   { return nil }
func (p ProtocolV2) OnBody(_ []byte) error      { return nil }
func (p ProtocolV2) OnMessageComplete() error   { return nil }

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

func divideBytes(src []byte, chunkSize int) [][]byte {
	var divided [][]byte

	for i := 0; i < len(src); i += chunkSize {
		end := i + chunkSize

		if end > len(src) {
			end = len(src)
		}

		divided = append(divided, src[i:end])
	}

	return divided
}

func BenchmarkSmallGETRequestBy1Char(b *testing.B) {
	protocol := ProtocolV2{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(smallGetRequest)
	}
}

func BenchmarkBigChromeRequestBy1Char(b *testing.B) {
	protocol := ProtocolV2{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(bigChromeRequest)
	}
}

func BenchmarkBigOwnRequest(b *testing.B) {
	protocol := ProtocolV2{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	req := []byte("POST /HelloWorldWhatsWrongHerewgfewggrrrrrrrrrrrrrrrrrrrrrrrrrrrrrrre HTTP/1.1\r\n" +
		"Header1: oergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerkhr\r\n" +
		"Header2: oergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerkhr\r\n" +
		"Header3: oergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerkhr\r\n" +
		"Header4: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header5: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header6: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header7: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header8: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header9: nergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohr\r\n" +
		"Header11: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header21: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header31: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header41: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header51: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header61: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header71: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header81: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header91: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header12: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header22: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header32: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgerohrk\r\n" +
		"Header42: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header52: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header62: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header72: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header82: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header92: ergegherkgeklrjgherkgherjkghrekerfgregregregregregrgergergergregehgkerhgkerjhgkrjehgernhro\r\n" +
		"Header112: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh ok\r\n" +
		"Header222: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh ok\r\n" +
		"Header312: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh ok\r\n" +
		"Header412: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Header512: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Header612: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Header712: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Header812: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Header912: erregherkgeklrjgherkgherjkghreerfgregregregregregrgergergergregekhgkerhgkerjhgkrjehgergh no\r\n" +
		"Host: thefavoriterushbyeveryone.is-a.dev\r\n" +
		"Content-Type: I love my dad I really really love my dad my dad is the best in the world wanna kiss him he is lapochka yesss better than lorem ipsum change my mind\r\n" +
		"content-length: 238" + // 238
		"\r\n\r\n" +
		"hey, dad! I love you. You must know it. You're the best person I ever knew. So can I kiss you? Regards, your lovely son" +
		"hey, dad! I love you. You must know it. You're the best person I ever knew. So can I kiss you? Regards, your lovely son")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		parser.Feed(req)
	}
}
