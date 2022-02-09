package httpparser

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/floordiv/snowdrop/httpparser"
)

type Protocol struct {
	Method         []byte
	Path           []byte
	Protocol       []byte
	Headers        map[string][]byte
	Body           []byte
	Completed      bool
	CompletedTimes int
}

func (p *Protocol) OnMessageBegin() error { return nil }

func (p *Protocol) OnMethod(method []byte) error {
	p.Method = method

	return nil
}

func (p *Protocol) OnPath(path []byte) error {
	p.Path = path

	return nil
}

func (p *Protocol) OnProtocol(proto []byte) error {
	p.Protocol = proto

	return nil
}

func (p *Protocol) OnHeadersBegin() error {
	p.Headers = make(map[string][]byte)

	return nil
}

func (p *Protocol) OnHeader(key, value []byte) error {
	p.Headers[string(key)] = value

	return nil
}

func (p *Protocol) OnHeadersComplete() error { return nil }

func (p *Protocol) OnBody(chunk []byte) error {
	p.Body = append(p.Body, chunk...)

	return nil
}

func (p *Protocol) OnMessageComplete() error {
	p.Completed = true
	p.CompletedTimes++

	return nil
}

func (p *Protocol) Clear() {
	p.Path = nil
	p.Method = nil
	p.Protocol = nil
	p.Headers = nil
	p.Body = nil
}

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
		if !strictHeadersCheck {
			key = strings.ToLower(key)
		}

		expectedValue, found := headers[key]

		if !strictHeadersCheck {
			expectedValue = strings.ToLower(expectedValue)
		}

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
		return false, fmt.Sprintf(`mismatching body: expected "%s", got %s`,
			expectBody, quote(protocol.Body))
	}

	return true, ""
}

func testOrdinaryGETRequestParse(t *testing.T, chunkSize int) {
	protocol := Protocol{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string{
		"Content-Type": "some content type",
		"Host":         "rush.dev",
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
		t.Error("no completion flag")
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
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
	testInvalidGETRequest(t, request, httpparser.ErrInvalidMethod)
}

func TestInvalidGETRequestEmptyMethod(t *testing.T) {
	request := []byte(" / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidMethod)
}

func TestInvalidGETRequestInvalidMethod(t *testing.T) {
	request := []byte("GETP / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidMethod)
}

func TestInvalidPOSTRequestExtraBody(t *testing.T) {
	request := []byte("POST / HTTP/1.1\r\nHost: rush.dev\r\nContent-Length: 13\r\n\r\nHello, world! Extra body")
	protocol := Protocol{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	err := FeedParser(parser, request, 5)

	if err == nil {
		t.Error("expected error, but no error was returned")
		return
	}
	if err != httpparser.ErrInvalidMethod {
		/*
			As we have a stream-based parser, we expect that extra-body always mean a new request
			So that's why we expect here InvalidMethod error: " Extra" is really invalid method
		*/

		t.Errorf(`expected InvalidMethod error, got "%s"`, err.Error())
		return
	}
}

func TestInvalidGETRequestUnknownProtocol(t *testing.T) {
	request := []byte("GET / HTTP/1.2\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrProtocolNotSupported)
}

func TestInvalidGETRequestEmptyPath(t *testing.T) {
	request := []byte("GET  HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidPath)
}

func TestInvalidGETRequestMissingPath(t *testing.T) {
	request := []byte("GET HTTP/1.2\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidPath)
}

func TestInvalidGETRequestInvalidHeader(t *testing.T) {
	request := []byte("GET / HTTP/1.1\r\nContent-Type some content type\r\nHost: rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidHeader)
}

func TestInvalidGETRequestNoSpaces(t *testing.T) {
	request := []byte("GET/HTTP/1.1\r\nContent-Typesomecontenttype\r\nHost:rush.dev\r\n\r\n")
	testInvalidGETRequest(t, request, httpparser.ErrInvalidMethod)
}

func testOrdinaryPOSTRequestParse(t *testing.T, chunkSize int) {
	protocol := Protocol{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	ordinaryGetRequest := []byte("POST / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev" +
		"\r\nContent-Length: 13\r\n\r\nHello, world!")

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string{
		"Content-Type":   "some content type",
		"Host":           "rush.dev",
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
		t.Error("no completion flag")
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	err := parser.Feed([]byte(request))

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string{
		"Host":            "localhost:8080",
		"Content-Type":    "some content type",
		"Accept-Encoding": "gzip, deflate, br",
	}
	bodyLenExpected := 0

	if err != nil {
		t.Errorf("error while parsing: %s", err)
	} else if !protocol.Completed {
		t.Errorf("no completion flag")
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
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	request := []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	err := FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag")
		return
	} else if err != nil {
		t.Errorf("got unexpected error: %s", err)
		return
	}

	err = FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag")
		return
	} else if err != nil {
		t.Errorf("got unexpected error: %s", err)
		return
	}
}

func TestConnectionClose(t *testing.T) {
	protocol := Protocol{}
	parser, _ := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	body := "Hello, I have a body for you!"
	request := []byte("POST / HTTP/1.1\r\nHost: rush.dev\r\nConnection: close\r\n\r\n" + body)
	err := FeedParser(parser, request, 5)

	if protocol.Completed {
		t.Error("got unexpected completion flag")
		return
	} else if err != nil {
		t.Error("unexpected error:", err.Error())
		return
	}

	// hope it's necessary to check method, path, protocol and headers after so many tests

	if !bytes.Equal(protocol.Body, []byte(body)) {
		t.Errorf("mismatching wanted and real body: wanted \"%s\", got %s", body, quote(protocol.Body))
		return
	}

	// on Connection: close header, the finish is connection close
	// in this case, reading from socket returns empty byte
	// and this will be a completion mark for our parser
	err = parser.Feed([]byte{})

	if !protocol.Completed {
		t.Error("no completion mark")
	} else if err != httpparser.ErrConnectionClosed {
		t.Error("expected ErrConnectionClosed error, got", err.Error())
	}
}
