package httpparser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/floordiv/snowdrop/src/httpparser"
)


type Protocol struct {
	Method 		[]byte
	Path 		[]byte
	Protocol 	[]byte
	Headers 	map[string]string
	Body 		[]byte
	Completed 	bool
}

func (p *Protocol) OnMessageBegin() {}

func (p *Protocol) OnMethod(method []byte) {
	p.Method = method
}

func (p *Protocol) OnPath(path []byte) {
	p.Path = path
}

func (p *Protocol) OnProtocol(proto []byte) {
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
}


func expect(
	protocol Protocol,
	expectedMethod, expectedPath, expectedProtocolString string,
	headers map[string]string,
	expectedBodyLength int,
	expectBody string,
	strictHeadersCheck bool) (succeeded bool, err string) {

	if string(protocol.Method) != expectedMethod {
		return false, fmt.Sprintf("expected method %s, got %s instead",
			string(protocol.Method), expectedMethod)
	}
	if string(protocol.Path) != expectedPath {
		return false, fmt.Sprintf("expected path %s, got %s instead",
			string(protocol.Path), expectedPath)
	}
	if string(protocol.Protocol) != expectedProtocolString {
		return false, fmt.Sprintf("expected protocol %s, got %s instead",
			string(protocol.Protocol), expectedProtocolString)
	}

	for key, value := range headers {
		if !strictHeadersCheck { key = strings.ToLower(key) }

		expectedValue, found := headers[key]

		if !strictHeadersCheck { expectedValue = strings.ToLower(expectedValue) }

		if !found {
			if strictHeadersCheck {
				return false, fmt.Sprintf("unexpected header: %s", key)
			} else {
				continue
			}
		}

		if expectedValue != value {
			return false, fmt.Sprintf("%s: values are mismatching (expected %s, got %s)",
				key, expectedValue, value)
		}
	}

	if expectedBodyLength >= 0 && len(protocol.Body) != expectedBodyLength {
		return false, fmt.Sprintf("mismatching body length: expected %d, got %d",
			expectedBodyLength, len(protocol.Body))
	} else if string(protocol.Body) != expectBody {
		return false, fmt.Sprintf("mismatching body: expected \"%s\", got \"%s\"",
			expectBody, string(protocol.Body))
	}

	return true, ""
}


func TestOrdinaryGETRequestParse(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Content-Type": "some content type",
		"Host": "rush.dev",
	}
	bodyLenExpected := 0

	ordinaryGetRequest := []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	err := FeedParser(parser, ordinaryGetRequest, 5)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
	} else if !protocol.Completed {
		t.Error("the whole request was fed to parser but he does not think so")
	}

	succeeded, errmsg := expect(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "", true)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestOrdinaryPOSTRequestParse(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)

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

	err := FeedParser(parser, ordinaryGetRequest, 5)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
	} else if !protocol.Completed {
		t.Error("the whole request was fed to parser but he does not think so")
	}

	succeeded, errmsg := expect(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "Hello, world!", true)

	if !succeeded {
		t.Error(errmsg)
	}
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
		"blqbu0Ksvtdr7q0iEns09m1D6MWlZv8JjB472GWnwfuDG; Goland-1dc491b=e03b2736-65ce-4ad0-b7ab-e4f8e1715c8b\r\n\r\n"

	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)
	completed, err := parser.Feed([]byte(request))

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
	} else if !completed {
		t.Errorf("the whole request was fed to parser but he does not think so")
	}

	succeeded, errmsg := expect(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, bodyLenExpected, "", false)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestChunkedTransferEncoding(t *testing.T) {
	request := "POST / HTTP/1.1\r\n" +
		"Content-Type: some content type\n\r" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n"

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Content-Type": "some content type",
		"Host": "rush.dev",
		"Transfer-Encoding": "chunked",
	}
	expectBody := "Hello, world!But what's wrong with you?Finally am here"

	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)
	completed, err := parser.Feed([]byte(request))

	if err != nil {
		t.Errorf("error while parsing: %s", err)
	} else if !completed {
		t.Errorf("the whole request was fed to parser but he does not think so")
	}

	succeeded, errmsg := expect(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, -1, expectBody, false)

	if !succeeded {
		t.Error(errmsg)
	}
}
