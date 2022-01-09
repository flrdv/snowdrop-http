package httpparser

import (
	"testing"

	httpparser "github.com/floordiv/snowdrop/src/snowdrop"
)

func TestParserReuseAbilityChunkedRequest(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)

	request := []byte("POST / HTTP/1.1\r\n" +
		"Content-Type: some content type\n\r" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	err := FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag before reuse")
		return
	} else if err != nil {
		t.Errorf("got unexpected error before reuse: %s", err)
		return
	}

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string {
		"Content-Type": "some content type",
		"Host": "rush.dev",
		"Transfer-Encoding": "chunked",
	}
	expectBody := "Hello, world!But what's wrong with you?Finally am here"

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, -1, expectBody, false)

	if !succeeded {
		t.Errorf("unexpected error before reuse: %s", errmsg)
		return
	}

	/*
		IDK why, but in some reason parser eats one \r\n after Transfer-Encoding header
		I really can't imagine why does it happens, and where

		But I lost 2 evenings to see this strange shit, so I don't mind
	*/
	request = []byte("POST / HTTP/1.1\r\n" +
		"Content-Type: some content type\n\r" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	err = FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("fed the whole request to parser, but no completion flag after reuse")
		return
	} else if err != nil {
		t.Errorf("got unexpected error after reuse: %s", err)
		return
	}

	succeeded, errmsg = want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, -1, expectBody, false)

	if !succeeded {
		t.Errorf("unexpected error after reuse: %s", errmsg)
		return
	}
}

func TestChunkedTransferEncodingFullRequestBody(t *testing.T) {
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
	parser := httpparser.NewHTTPRequestParser(&protocol, BufferLength)
	err := parser.Feed([]byte(request))

	if err != nil {
		t.Errorf("error while parsing: %s", err)
		return
	} else if !protocol.Completed {
		t.Errorf("the whole request was fed to parser but he does not think so")
		return
	}

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, -1, expectBody, false)

	if !succeeded {
		t.Error(errmsg)
	}
}
