package httpparser

import (
	"github.com/floordiv/snowdrop/src/httpparser"
	"testing"
)


type Protocol struct {
	Method []byte
	Path []byte
	Protocol []byte
	Headers map[string]string
	Body []byte
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

func (p *Protocol) OnMessageComplete() {}


func TestOrdinaryGETRequestParse(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol)

	methodExpected := "GET"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	contentTypeHeaderExpected := "some content type"
	hostHeaderExpected := "rush.dev"
	bodyLenExpected := 0

	ordinaryGetRequest := []byte("GET / HTTP/1.1\r\nContent-Type: some content type\r\nHost: rush.dev\r\n\r\n")
	err := FeedParser(parser, ordinaryGetRequest, 5)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
	}

	if string(protocol.Method) != methodExpected {
		t.Errorf("parsed method is invalid: expected %s, got %s", methodExpected, string(protocol.Method))
	}
	if string(protocol.Path) != pathExpected {
		t.Errorf("parsed path is invalid: expected %s, got %s", pathExpected, string(protocol.Path))
	}
	if string(protocol.Protocol) != protocolExpected {
		t.Errorf("parsed protocol is invalid: expected %s, got %s", protocolExpected, string(protocol.Protocol))
	}
	if protocol.Headers["Content-Type"] != contentTypeHeaderExpected {
		t.Errorf("parsed Content-Type header is invalid: expected %s, got %s",
			contentTypeHeaderExpected, protocol.Headers["Content-Type"])
	}
	if protocol.Headers["Host"] != hostHeaderExpected {
		t.Errorf("parsed Host header is invalid: expected %s, got %s",
			hostHeaderExpected, protocol.Headers["Host"])
	}
	if len(protocol.Body) != bodyLenExpected {
		t.Errorf("parsed body is invalid: expected nothing, got \"%s\"", string(protocol.Body))
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
	contentTypeHeaderExpected := "some content type"
	hostHeaderExpected := "rush.dev"
	contentLengthHeaderExpected := "13"
	bodyLenExpected := 13

	err := FeedParser(parser, ordinaryGetRequest, 5)

	if err != nil {
		t.Errorf("parser returned error: %s\n", err)
	}

	if string(protocol.Method) != methodExpected {
		t.Errorf("parsed method is invalid: expected %s, got %s", methodExpected, string(protocol.Method))
	}
	if string(protocol.Path) != pathExpected {
		t.Errorf("parsed path is invalid: expected %s, got %s", pathExpected, string(protocol.Path))
	}
	if string(protocol.Protocol) != protocolExpected {
		t.Errorf("parsed protocol is invalid: expected %s, got %s", protocolExpected, string(protocol.Protocol))
	}
	if protocol.Headers["Content-Type"] != contentTypeHeaderExpected {
		t.Errorf("parsed Content-Type header is invalid: expected %s, got %s",
			contentTypeHeaderExpected, protocol.Headers["Content-Type"])
	}
	if protocol.Headers["Host"] != hostHeaderExpected {
		t.Errorf("parsed Host header is invalid: expected %s, got %s",
			hostHeaderExpected, protocol.Headers["Host"])
	}
	if protocol.Headers["Content-Length"] != contentLengthHeaderExpected {
		t.Errorf("parsed Content-Length header is invalid: expected %s, got %s",
			hostHeaderExpected, protocol.Headers["Content-Length"])
	}
	if len(protocol.Body) != bodyLenExpected {
		t.Errorf("parsed body is invalid: expected nothing, got \"%s\"", string(protocol.Body))
	}
}
