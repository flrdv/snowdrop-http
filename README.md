# SnowDrop
A LLHTTP-like HTTP requests parser. As there are no external HTTP parsers in golang (excluding wildcat, but last time it was contributed was 2014), I decided to do it by my own.

# Simple usage:
This parser is inspired by LLHTTP, as I said before. So, you need to implement a protocol, which is a usual struct, but needs to implement such a methods:

```golang
type Protocol interface {
	OnMessageBegin()
	OnMethod(method []byte)
	OnPath(path []byte)
	OnProtocol(protocol []byte)
	OnHeadersBegin()
	OnHeader(key, value []byte)
	OnHeadersComplete()
	OnBody(chunk []byte)
	OnMessageComplete()
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

This settings are passed to parser ALWAYS. It may be even not specified as parser will set unspecified values with default ones. If buffers aren't specified, they will be allocated automatically. All this stuff you can find in [src/httpparser/settings.go](https://github.com/fakefloordiv/snowdrop-http/blob/master/src/httpparser/settings.go)

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
> *A*: Currently this feature is in TODO, but soon I will implement it. No, currently callback can not return error

# Example:

```golang
package main

import (
	"github.com/floordiv/snowdrop-http/src/httpparser"
)


type Protocol struct {
	// implement httpparser.IProtocol here
}


func main() {
	protocol := Protocol{...}
	parser := httpparser.NewHTTPRequestParser(&protocol, httpparser.Settings{})
	data := ... // http request taken from any source, with []byte type
	err := parser.Feed(data)
	
	if err != nil {
		// parser isn't able to parse anymore
		log.Fatal(err)
	}
	
	// that's it! Now everything has been processed with your protocol
}
```
