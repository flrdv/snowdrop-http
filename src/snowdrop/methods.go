package snowdrop


type HTTPMethod string

var (
	GET HTTPMethod = "GET"
	HEAD HTTPMethod = "HEAD"
	POST HTTPMethod = "POST"
	PUT HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
	CONNECT HTTPMethod = "CONNECT"
	OPTIONS HTTPMethod = "OPTIONS"
	TRACE HTTPMethod = "TRACE"
	PATCH HTTPMethod = "PATCH"
)

func IsMethodValid(method string) bool {
	switch HTTPMethod(method) {
	case GET,
		 HEAD,
		 POST,
		 PUT,
		 DELETE,
		 CONNECT,
		 OPTIONS,
		 TRACE,
		 PATCH: return true
	}

	return false
}
