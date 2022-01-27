# SnowDrop
A LLHTTP-like HTTP requests parser. As there are no external HTTP parsers in golang (excluding wildcat, but last time it was contributed was 2014), I decided to do it by my own.

# Important
Currently there are 2 implementations, both can be found in folder src/. `httpparser` is the first one, `snowdrop` is v2. v2 is faster than v1, but only if request isn't chunked (and can be parsed for one `parser.Feed()` call). But still has linear (or even squared) big O. First version is slower a bit, but has more linear allocs and time values. So currently this parser isn't recommended as prod-ready cause of awful speed & big O, but I have a hope that this text will disappear in future

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

> *A*: OnBody() callback will be called each time when full chunk was received

<br>

> *Q*: How does parser behave in case of extra-bytes are passed?

> *A*: Parser is stream-based, so parser's lifetime equals to connection lifetime. Protocol methods will be called in a loop, every time, when new data is ready to be pushed

<br>

> *Q*: Can it parse requests that use not CRLF, but just LF?

> *A*: Yes. Parser may parse even requests with mixed usage of CRLF and LF

<br>

> *Q*: What's if it is not a GET request, but Content-Length is not specified?

> *A*: Request's body will be marked as empty (QA below referrs to this question)

<br>

> *Q*: How will parser behave in case of empty request body?

> *A*: OnBody() won't be called during parsing, but OnMessageComplete() will

<br>

> *Q*: What will happen if an error will occur?

> *A*: Parser will die (the state of it will be set to Dead forever), and all the attempts to feed it again will return error ParserIsDead

# Example:

```golang
package main

import (
	httpparser "github.com/floordiv/snowdrop-http/src/snowdrop"
)


type Protocol struct {
	// implement httpparser.IProtocol here
}


func main() {
	protocol := Protocol{...}
	bufferLength := 65535
	parser := httpparser.NewHTTPRequestParser(&protocol, bufferLength)
	data := ... // http request taken from any source, with []byte type
	err := parser.Feed(data)
	
	if err != nil {
		// parser isn't able to parse anymore
		log.Fatal(err)
	}
	
	// that's it! Now everything has been processed with your protocol
}
```
