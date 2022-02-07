package httpparser

import (
	"github.com/floordiv/snowdrop/src/httpparser"
	"testing"
)

func TestParserReuseAbilityChunkedRequest(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})

	request := []byte("POST / HTTP/1.1\r\n" +
		"Content-Type: some content type\r\n" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	err := FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("no completion flag")
		return
	} else if err != nil {
		t.Errorf("got unexpected error before reuse: %s", err)
		return
	}

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string{
		"Content-Type":      "some content type",
		"Host":              "rush.dev",
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
		"Content-Type: some content type\r\n" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	protocol.Clear()
	err = FeedParser(parser, request, 5)

	if !protocol.Completed {
		t.Error("no completion flag after reuse")
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
		"Content-Type: some content type\r\n" +
		"Host: rush.dev\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"\r\nd\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n"

	methodExpected := "POST"
	pathExpected := "/"
	protocolExpected := "HTTP/1.1"
	headersExpected := map[string]string{
		"Content-Type":      "some content type",
		"Host":              "rush.dev",
		"Transfer-Encoding": "chunked",
	}
	expectBody := "Hello, world!But what's wrong with you?Finally am here"

	protocol := Protocol{}
	parser := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	err := parser.Feed([]byte(request))

	if err != nil {
		t.Errorf("error while parsing: %s", err)
		return
	} else if !protocol.Completed {
		t.Errorf("no completion flag")
		return
	}

	succeeded, errmsg := want(protocol,
		methodExpected, pathExpected, protocolExpected,
		headersExpected, -1, expectBody, false)

	if !succeeded {
		t.Error(errmsg)
	}
}

func TestChunkOverflow(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewChunkedBodyParser(protocol.OnBody, 65535)
	data := []byte("d\r\nHello, world! Overflow here\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	done, _, err := parser.Feed(data)

	if !done {
		t.Errorf("no completion flag")
		return
	}

	if err == nil {
		t.Errorf("no error returned")
		return
	}

	if err != httpparser.InvalidChunkSplitter {
		t.Errorf(`expected InvalidChunkSplitter error, got msg="%s"`, err.Error())
		return
	}
}

func TestChunkTooSmall(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewChunkedBodyParser(protocol.OnBody, 65535)
	data := []byte("d\r\nHello, ...\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	done, _, err := parser.Feed(data)

	if !done {
		t.Errorf("no completion flag")
		return
	}

	if err == nil {
		t.Errorf("no error returned")
		return
	}

	if err != httpparser.InvalidChunkSplitter {
		t.Errorf(`expected InvalidChunkSplitter error, got msg="%s"`, err.Error())
		return
	}
}

func TestMixChunkSplitters(t *testing.T) {
	protocol := Protocol{}
	parser := httpparser.NewChunkedBodyParser(protocol.OnBody, 65535)
	data := []byte("d\r\nHello, world!\n1a\r\nBut what's wrong with you?\nf\nFinally am here\r\n0\r\n\n")

	done, _, err := parser.Feed(data)

	if !done {
		t.Error("no completion flag")
		return
	}

	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}
}

func TestWithDifferentBlockSizes(t *testing.T) {
	protocol := Protocol{}
	data := []byte("d\r\nHello, world!\r\n1a\r\nBut what's wrong with you?\r\nf\r\nFinally am here\r\n0\r\n\r\n")

	for i := 1; i <= len(data); i++ {
		parser := httpparser.NewChunkedBodyParser(protocol.OnBody, 65535)

		for j := 0; j < len(data); j += i {
			end := j + i

			if end > len(data) {
				end = len(data)
			}

			done, _, err := parser.Feed(data[j:end])

			if err != nil {
				t.Errorf("unexpected error: %s", err.Error())
				return
			}

			if done && end < len(data) {
				t.Error("having completion flag, but it isn't really completed")
				return
			}
		}
	}
}
