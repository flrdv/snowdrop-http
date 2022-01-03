package httpparser

type iparser interface{
	Feed([]byte) (bool, []byte, error)
}

func FeedParser(parser iparser, data []byte, chunksSize int) (completed bool, err error) {
	for i := 0; i < len(data); i += chunksSize {
		end := i + chunksSize

		if end > len(data) {
			end = len(data)
		}

		completed, _, err  = parser.Feed(data[i:end])

		if err != nil {
			return completed, err
		}

		if completed {
			return true, nil
		}
	}

	return false, nil
}
