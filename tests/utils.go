package httpparser

type iparser interface{
	Feed([]byte) error
}

func FeedParser(parser iparser, data []byte, chunksSize int) error {
	for i := 0; i < len(data); i += chunksSize {
		end := i + chunksSize

		if end > len(data) {
			end = len(data)
		}

		rawPiece := data[i:end]
		piece := make([]byte, len(rawPiece))
		copy(piece, rawPiece)

		err := parser.Feed(piece)

		if err != nil {
			return err
		}
	}

	return nil
}
