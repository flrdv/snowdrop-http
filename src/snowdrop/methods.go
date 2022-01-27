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

var HTTPMethods = [][]byte{GET, HEAD, POST, PUT, DELETE, CONNECT, OPTIONS, TRACE, PATCH}

func IsMethodValid(method []byte) bool {
	for _, existingMethod := range HTTPMethods {
		if bytes.Equal(existingMethod, method) {
			return true
		}
	}

	return false
}
