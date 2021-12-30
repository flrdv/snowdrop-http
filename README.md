# SnowDrop
A LLHTTP-like HTTP requests parser. As there are no external HTTP parsers in golang (excluding wildcat, but last time it was contributed was 2014), I decided to do it by my own.


# Simple usage:
This parser is inspired by LLHTTP, as I said before. So, you need to implement a protocol, which is a usual struct, but needs to implement such a methods:

```
type Protocol interface {
	OnMessageBegin()
	OnMethod(method []byte)
	OnPath(path []byte)
	OnProtocol(protocol []byte)
	OnHeadersBegin()
	OnHeader(key, value string)
	OnHeadersComplete()
	OnBody(chunk []byte)
	OnMessageComplete()
}
```

Example of implemented protocol you can find [here](https://github.com/floordiv/snowdrop-http/blob/master/tests/parser_test.go#L9)

# FAQ
> *Q*: How does parser behave in case of chunked request?

> *A*: OnBody() callback will be called. There are no guarantees that the whole chunk will be passed once, but may be only a part of actual chunk, depends on how much data will be fed

<br>

> *Q*: How does parser behave in case of extra-bytes are passed?

> *A*: They'll be just ignored

<br>

> *Q*: Can it parse requests that use not CRLF, but just LF?

> *A*: Yes. Parser is built that it ignores all the \r characters (you should keep this feature in your mind as this is possibly security issue), so it parses CRLF sequences as well as LF

<br>

> *Q*: What's if it is not a GET request, but Content-Length is not specified?

> *A*: Body won't be parsed, parser will think this request has empty body

<br>

> *Q*: How will parser behave in case of empty request body?

> *A*: OnBody() won't be called during parsing, but OnMessageComplete() will

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
	parser := httpparser.NewHTTPRequestParser(&protocol)
	data := ... // http request taken from any source, but []byte
	completed, err := parser.Feed(data)
	
	if err != nil {
		log.Fatal(err)
	}
	
	// that's it! Now everything has been processed with your protocol
}
```
