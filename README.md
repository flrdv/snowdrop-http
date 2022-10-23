# Warning!
This repository is unmaintained anymore. Parser has multiple performance issues and bugs that must be fixed, so if you gonna use, please open an issue 

# SnowDrop
A LLHTTP-like HTTP requests parser. As there are no external HTTP parsers in golang (excluding wildcat, but it doesn't support stream-based parsing and last time it was contributed is 2014), so I decided to do it by my own as I need it for my web-server. The decision to do parser and external package follows the idea that it'll be more clear to divide web-server and http parser, but also to let other people use it in their projects if they need it. 

# Simple usage:
This parser is inspired by LLHTTP, as I said before. So, you need to implement a protocol, which is a usual struct, but needs to implement such a methods:

```golang
type Protocol interface {
	OnMessageBegin() error
	OnMethod([]byte) error
	OnPath([]byte) error
	OnProtocol([]byte) error
	OnHeadersBegin() error
	OnHeader([]byte, []byte) error
	OnHeadersComplete() error
	OnBody([]byte) error
	OnMessageComplete() error
}
```

Example of implemented protocol you can find [here](https://github.com/floordiv/snowdrop-http/blob/master/tests/parser_test.go#L12)

Also parser has settings structure:

```golang
type Settings struct {
	// hard limits
	MaxPathLength       int
	MaxHeaderLineLength int
	MaxBodyLength       int
	MaxChunkLength      int

	// soft limits
	InitialPathBufferLength    int
	InitialHeadersBufferLength int

	StartLineBuffer []byte
	HeadersBuffer   []byte
}
```

This settings are passed to parser ALWAYS. It may be even not specified as parser will set unspecified values with default ones. If buffers aren't specified, they will be allocated automatically. All this stuff you can find in [httpparser/settings.go](https://github.com/fakefloordiv/snowdrop-http/blob/master/httpparser/settings.go)

# FAQ
> *Q*: How does parser behave in case of chunked request?

> *A*: OnBody() callback will be called each time when a piece of body was received. It may be even one single byte

<br>

> *Q*: How does parser behave in case of extra-bytes are passed?

> *A*: Parser is stream-based, so parser's lifetime equals to connection lifetime. This means that extra-bytes will be parsed as a beginning of the next request

<br>

> *Q*: Can it parse requests that use not CRLF, but just LF?

> *A*: Yes. Parser can parse even requests with mixed usage of CRLF and LF

<br>

> *Q*: What's if it is not a GET request, but Content-Length is not specified?

> *A*: Request's body will be marked as empty (QA below referrs to this question), if "Connection" header is not set to closed (in this case, request body will be parsed until empty bytes array will be passed as a food)

<br>

> *Q*: How will parser behave in case of empty request body?

> *A*: OnBody() will be never called, but OnMessageComplete() will

<br>

> *Q*: What will happen if an error will occur?

> *A*: Parser will die (the state of it will be set to "dead" forever), and all the attempts to feed it again will return ParserIsDead error

<br>

> *Q*: What's if an error occurred in protocol callback? 

> *A*: Parser will die and return error from callback to server, BUT in case `httpparser.Upgrade` struct is returned from `OnMessageComplete()`, server won't die, but will just return the error to http server. Warning: in case `httpparser.Upgrade` will be returned from any other callback, this won't work and parser will die anyway

<br>

> *Q*: What's if we have a simple request that doesn't even contains headers, for example, `GET / HTTP/1.1\r\n\r\n`?

> *A*: There are 7 obligatory callbacks that are guarantateed to be called (if no errors occurred): `OnMessageBegin`, `OnMethod`, `OnPath`, `OnProtocol`, `OnHeadersBegin`, `OnHeadersComplete`, `OnMessageComplete`. So all them will be called during parsing ANY request except invalid ones

# Example:

```golang
package main

import (
	"github.com/floordiv/snowdrop-http/src/httpparser"
)


type MyProtocol struct {
	// implement httpparser.Protocol here
}


func main() {
	protocol := MyProtocol{...}
	parser := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	data := ... // http request taken from any source, with []byte type
	
	if err := parser.Feed(data); err != nil {
		// parser isn't able to parse anymore
		log.Fatal(err)
	}
	
	// that's it! Now everything has been processed with your protocol
}
```

Parser also can return errors:
- `ErrInvalidMethod`      
- `ErrInvalidPath`
- `ErrProtocolNotSupported`
- `ErrInvalidHeader`
- `ErrBufferOverflow`
- `ErrInvalidContentLength`
- `ErrRequestSyntaxError`
- `ErrBodyTooBig`
- `ErrTooBigChunkSize`
- `ErrInvalidChunkSize`
- `ErrInvalidChunkSplitter`
- `ErrConnectionClosed`
- `ErrParserIsDead`

Important: this is not a finite list of errors may be returned by parser. In case of errors returned from callbacks, parser will die and return error from callback

Also `httpparser.Upgrade` struct may be returned as an error. It can be constructed from `httpparser.NewUpgrade(string)` function

Details about errors you can find in [httpparser/errors.go](https://github.com/fakefloordiv/snowdrop-http/blob/master/httpparser/errors.go)
