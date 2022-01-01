package httpparser

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
	case eq(method, GET): 		return true
	case eq(method, HEAD): 		return true
	case eq(method, POST): 		return true
	case eq(method, PUT):		return true
	case eq(method, DELETE):	return true
	case eq(method, CONNECT):	return true
	case eq(method, OPTIONS):	return true
	case eq(method, TRACE):		return true
	case eq(method, PATCH): 	return true
	}

	return false
}
