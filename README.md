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

Just like LLHTTP, OnBody() callback receives a part of body that was received. In case of chunked requests, OnBody() may receive not the whole chunk, but a part, depending on 
whether the whole chunk will be fed to the parser. But guaranteed that OnBody() won't receive more than whole chunk per once

# Example:

```
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
