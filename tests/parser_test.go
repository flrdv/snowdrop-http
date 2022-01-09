package httpparser

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	httpparser "github.com/floordiv/snowdrop/src/snowdrop"
)


const BufferLength = 65535

type Protocol struct {
	Method 			string
	Path 			string
	Protocol 		string
	Headers 		map[string]string
	Body 			[]byte
	Completed 		bool
	CompletedTimes  int
}

func (p *Protocol) OnMessageBegin() {}

func (p *Protocol) OnMethod(method string) {
	p.Method = method
}

func (p *Protocol) OnPath(path string) {
	p.Path = path
}

func (p *Protocol) OnProtocol(proto string) {
	p.Protocol = proto
}

func (p *Protocol) OnHeadersBegin() {
	p.Headers = make(map[string]string)
}

func (p *Protocol) OnHeader(key, value string) {
	p.Headers[key] = value
}

func (p *Protocol) OnHeadersComplete() {}

func (p *Protocol) OnBody(chunk []byte) {
	p.Body = append(p.Body, chunk...)
}

func (p *Protocol) OnMessageComplete() {
	p.Completed = true
	p.CompletedTimes++
}

/*
And this protocol is for benchmarking, that's why it's empty - to avoid
extra-overhead
 */
type BenchProtocol struct {}

func (p *BenchProtocol) OnMessageBegin() {}

func (p *BenchProtocol) OnMethod(_ []byte) {}

func (p *BenchProtocol) OnPath(_ []byte) {}

func (p *BenchProtocol) OnProtocol(_ []byte) {}

func (p *BenchProtocol) OnHeadersBegin() {}

func (p *BenchProtocol) OnHeader(_, _ []byte) {}

func (p *BenchProtocol) OnHeadersComplete() {}

func (p *BenchProtocol) OnBody(_ []byte) {}

func (p *BenchProtocol) OnMessageComplete() {}

func quote(data []byte) string {
	return strconv.Quote(string(data))
}

func want(
	protocol Protocol,
	expectedMethod, expectedPath, expectedProtocolString string,
	headers map[string]string,
	expectedBodyLength int,
	expectBody string,
	strictHeadersCheck bool) (succeeded bool, err string) {

	if string(protocol.Method) != expectedMethod {
		return false, fmt.Sprintf("expected method %s, got %s instead",
			expectedMethod, protocol.Method)
	}
	if string(protocol.Path) != expectedPath {
		return false, fmt.Sprintf("expected path %s, got %s instead",
			expectedPath, protocol.Path)
	}
	if string(protocol.Protocol) != expectedProtocolString {
		return false, fmt.Sprintf("expected protocol %s, got %s instead",
			expectedProtocolString, protocol.Protocol)
	}

	for key, value := range headers {
		if !strictHeadersCheck { key = strings.ToLower(key) }

		expectedValue, found := headers[key]

		if !strictHeadersCheck { expectedValue = strings.ToLower(expectedValue) }

		if !found {
			if strictHeadersCheck {
				return false, fmt.Sprintf("unexpected header: %s", strconv.Quote(key))
			} else {
				continue
			}
		}

		if expectedValue != value {
			return false, fmt.Sprintf("%s: values are mismatching (expected %s, got %s)",
				strconv.Quote(key), strconv.Quote(expectedValue), strconv.Quote(value))
		}
	}

	if expectedBodyLength >= 0 && len(protocol.Body) != expectedBodyLength {
		return false, fmt.Sprintf("mismatching body length: expected %d, got %d",
			expectedBodyLength, len(protocol.Body))
	} else if string(protocol.Body) != expectBody {
		return false, fmt.Sprintf(`mismatching body: expected "%s", got "%s"`,
			expectBody, quote(protocol.Body))
	}

	return true, ""
}

func testOrdinaryGETRequestParse(t *testing.T, chunkSize int) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Content-Type": "some content type",
		"Host": "rush.dev",
	}
	bodyLenExpected := 0

	ordinaryGetRequest := []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")

	if chunkSize == -1 {
		chunkSize = len(ordinaryGetRequest) + 1
	}

	err := FeedParser(parser, ordinaryGetRequest, chunkSize)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
		return
	} else if protocol.CompletedTimes != 1 {
		t.Error("the whole request was fed to parser but he does not think so")
		return
	}

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "", true)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestOrdinaryGETRequestParse1Char(t *testing.T) {
	testOrdinaryGETRequestParse(t, 1)
}

func TestOrdinaryGETRequestParse2Chars(t *testing.T) {
	testOrdinaryGETRequestParse(t, 2)
}

func TestOrdinaryGETRequestParse5Chars(t *testing.T) {
	testOrdinaryGETRequestParse(t, 5)
}

func TestOrdinaryGETRequestParseFull(t *testing.T) {
	testOrdinaryGETRequestParse(t, -1)
}

func testInvalidGETRequest(t *testing.T, request []byte, errorWanted error) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)
	err := FeedParser(parser, request, 5)

	if err != nil && err != errorWanted {
		t.Errorf(`expected "%s" error, got "%s" instead`, errorWanted, err)
	} else if err == nil && protocol.CompletedTimes != 1 {
		t.Error("parser didn't return an error and didn't mark request as completed")
	} else if err == nil {
		t.Error("parser didn't return an error")
	}
}

func TestInvalidGETRequestMissingMethod(t *testing.T) {
	request := []byte("/ HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.RequestSyntaxError)
}

func TestInvalidPOSTRequestExtraBody(t *testing.T) {
	request := []byte("POST / HTTP/1.1\r\nHost: rush.dev\r\nContent-Length: 13\r\n\r\nHello, world! Extra body")
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)
	err := FeedParser(parser, request, 5)

	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	} else if !protocol.Completed {
		t.Error("fed the whole request (and even more) but no completion mark")
		return
	}

	wantedMethod := "POST"
	wantedPath := "/"
	wantedProtocol := "HTTP/1.1"
	wantedHeaders := map[string]string {
		"Host": "rush.dev",
	}
	bodyLenWanted := 13

	succeeded, errmsg := want(
		protocol, wantedMethod, wantedPath,
		wantedProtocol, wantedHeaders, bodyLenWanted,
		"Hello, world!", false)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestInvalidGETRequestUnknownProtocol(t *testing.T) {
	request := []byte("GET / HTTP/1.2\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.InvalidRequestData)
}

func TestInvalidGETRequestEmptyPath(t *testing.T) {
	request := []byte("GET  HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.InvalidRequestData)
}

func TestInvalidGETRequestMissingPath(t *testing.T) {
	request := []byte("GET HTTP/1.2\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.RequestSyntaxError)
}

func TestInvalidGETRequestInvalidHeader(t *testing.T) {
	request := []byte("GET / HTTP/1.1\r\nContent-Type some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.InvalidHeader)
}

func TestInvalidGETRequestNoSpaces(t *testing.T) {
	request := []byte("GET/HTTP/1.1\r\nContent-Typesomecontenttype\r\nHost:rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.RequestSyntaxError)
}

func testOrdinaryPOSTRequestParse(t *testing.T, chunkSize int) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)

	ordinaryGetRequest := []byte("POST / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev" +
		"\r\nContent-Length: 13\r\n\r\nHello, world!")

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Content-Type": "some content type",
		"Host": "rush.dev",
		"Content-Length": "13",
	}
	bodyLenExpected := 13

	if chunkSize == -1 {
		chunkSize = len(ordinaryGetRequest)
	}

	err := FeedParser(parser, ordinaryGetRequest, chunkSize)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
		return
	} else if !protocol.Completed {
		t.Error("the whole request was fed to parser but he does not think so")
		return
	}

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "Hello, world!", true)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestOrdinaryPOSTRequestParse1Char(t *testing.T) {
	testOrdinaryPOSTRequestParse(t, 1)
}

func TestOrdinaryPOSTRequestParse2Chars(t *testing.T) {
	testOrdinaryPOSTRequestParse(t, 2)
}

func TestOrdinaryPOSTRequestParse5Chars(t *testing.T) {
	testOrdinaryPOSTRequestParse(t, 5)
}

func TestOrdinaryPOSTRequestParseFull(t *testing.T) {
	testOrdinaryPOSTRequestParse(t, -1)
}

func TestChromeGETRequest(t *testing.T) {
	request := "GET / HTTP/1.1\r\nHost: localhost:8080\r\nConnection: keep-alive\r\nCache-Control: max-age=0" +
		"\r\nsec-ch-ua: \" Not A;Brand\";v=\"99\", \"Chromium\";v=\"96\", \"Google Chrome\";v=\"96\"" +
		"\r\nsec-ch-ua-mobile: ?0\r\nsec-ch-ua-platform: \"Windows\"\r\nUpgrade-Insecure-Requests: 1" +
		"\r\nUser-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96" +
		".0.4664.110 Safari/537.36\r\n" +
		"Accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8," +
		"application/signed-exchange;v=b3;q=0.9\r\nSec-Fetch-Site: none\r\nSec-Fetch-Mode: navigate" +
		"\r\nSec-Fetch-User: ?1\r\nSec-Fetch-Dest: document\r\nAccept-Encoding: gzip, deflate, br" +
		"\r\nAccept-Language: ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7,uk;q=0.6\r\nCookie: csrftoken=y1y3SinAMbiYy7yn9Oc" +
		"blqbudgdgdgdgddgdgdgdgdsgsdgsdgdgddgdGWnwfuDG; Goland-1dc491b=e03b2dgdgvdfgad0-b7ab-e4f8e1715c8b\r\n\r\n"

	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)
	err := parser.Feed([]byte(request))

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Host": "localhost:8080",
		"Content-Type": "some content type",
		"Accept-Encoding": "gzip, deflate, br",
	}
	bodyLenExpected := 0

	if err != nil {
		t.Errorf("error while parsing: %s", err)
	} else if !protocol.Completed {
		t.Errorf("the whole request was fed to parser but he does not think so")
	}

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "", false)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestParserReuseAbility(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)

	request := []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	err := FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag")
		return
	} else if err != nil {
		t.Errorf("got unexpected error: %s", err)
		return
	}

	protocol = Protocol{}
	err = FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag")
		return
	} else if err != nil {
		t.Errorf("got unexpected error: %s", err)
		return
	}
}
