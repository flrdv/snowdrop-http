package httpparser


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
