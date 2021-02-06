package buffbits

type errReadWriter struct {
	err error
}

func (erw errReadWriter) Write(b []byte) (int, error) {
	return len(b), erw.err
}

func (erw errReadWriter) Read(p []byte) (int, error) {
	return 0, erw.err
}
