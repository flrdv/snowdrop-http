package httpparser

import (
	"github.com/floordiv/snowdrop/src/httpparser"
)


func FeedParser(parser *httpparser.HTTPRequestParser, data []byte, chunksSize int) error {
	for i := 0; i < len(data); i += chunksSize {
		end := i + chunksSize

		if end > len(data) {
			end = len(data)
		}

		completed, err := parser.Feed(data[i:end])

		if err != nil {
			return err
		}

		if completed {
			return nil
		}
	}

	return nil
}
