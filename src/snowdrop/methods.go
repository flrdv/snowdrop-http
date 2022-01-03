package snowdrop

import "bytes"


type HTTPMethod []byte

var (
	GET HTTPMethod = []byte("GET")
	HEAD HTTPMethod = []byte("HEAD")
	POST HTTPMethod = []byte("POST")
	PUT HTTPMethod = []byte("PUT")
	DELETE HTTPMethod = []byte("DELETE")
	CONNECT HTTPMethod = []byte("CONNECT")
	OPTIONS HTTPMethod = []byte("OPTIONS")
	TRACE HTTPMethod = []byte("TRACE")
	PATCH HTTPMethod = []byte("PATCH")
)

func IsMethodValid(method []byte) bool {
	eq := bytes.Equal

	switch {
	case eq(method, GET),
		 eq(method, HEAD),
		 eq(method, POST),
		 eq(method, PUT),
		 eq(method, DELETE),
		 eq(method, CONNECT),
		 eq(method, OPTIONS),
		 eq(method, TRACE),
		 eq(method, PATCH): return true
	}

	return false
}
