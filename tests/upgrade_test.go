package httpparser

import (
	"fmt"
	"testing"

	"github.com/fakefloordiv/snowdrop-http/httpparser"
)

const debug = false

func Println(elements ...string) {
	if debug {
		fmt.Println(elements)
	}
}

type protocol struct {
}

func (p protocol) OnMessageBegin() error {
	Println("OnMessageBegin")

	return nil
}

func (p protocol) OnMethod(bytes []byte) error {
	Println("OnMethod", string(bytes))

	return nil
}

func (p protocol) OnPath(bytes []byte) error {
	Println("OnPath", string(bytes))

	return nil
}

func (p protocol) OnProtocol(bytes []byte) error {
	Println("OnProtocol", string(bytes))

	return nil
}

func (p protocol) OnHeadersBegin() error {
	Println("OnHeadersBegin")

	return nil
}

func (p protocol) OnHeader(bytes []byte, bytes2 []byte) error {
	Println("OnHeader", string(bytes), string(bytes2))

	return nil
}

func (p protocol) OnHeadersComplete() error {
	Println("OnHeadersComplete")

	return nil
}

func (p protocol) OnBody(bytes []byte) error {
	Println("OnBody", string(bytes))

	return nil
}

func (p protocol) OnMessageComplete() error {
	Println("OnMessageComplete")

	return httpparser.NewUpgrade("http/2, http/3")
}

func TestUpgrade(t *testing.T) {
	parser, err := httpparser.NewHTTPRequestParser(&protocol{}, httpparser.Settings{})

	if err != nil {
		t.Fatal(err)
		return
	}

	request := []byte("GET / HTTP/1.1\nConnection: upgrade\nUpgrade: http/2, http/3\n\n")
	err = parser.Feed(request)
	switch err.(type) {
	case httpparser.Upgrade:
	default:
		t.Fatal("Wanted Upgrade, got", err)
		return
	}

	if err = parser.Feed(nil); err != nil {
		// If we pass an empty or nil slice, nil must be returned if parser
		// isn't dead or Connection header isn't set to Close. Otherwise,
		// nil must be returned
		t.Fatal("Wanted nil, got", err)
	}
}
