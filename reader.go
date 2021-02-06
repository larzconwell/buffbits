package buffbits

import (
	"bufio"
	"errors"
	"io"
)

// Reader implements buffered bit level read access from an underlying io.Reader.
// Buffered reading is powered by the bufio package. If an error occurs while reading
// from a Reader, no more data will be read and all subsequent calls will return an
// error.
type Reader struct {
	br    *bufio.Reader
	buf   uint64
	count int
	err   error
}

// NewReader creates a buffered bit reader reading from r.
func NewReader(r io.Reader) *Reader {
	return &Reader{br: bufio.NewReader(r)}
}

// NewReaderSize creates a buffered bit reader reading from r using a buffer of size bytes.
func NewReaderSize(r io.Reader, size int) *Reader {
	return &Reader{br: bufio.NewReaderSize(r, size)}
}

// Err returns the first error that was encountered by the Reader.
func (r *Reader) Err() error {
	return r.err
}

// Reset discards any state and switches reading from the provided reader.
func (r *Reader) Reset(reader io.Reader) {
	r.br.Reset(reader)
	r.buf = 0
	r.count = 0
	r.err = nil
}

// Read reads count bits from the Reader and returns them in the lower positions of
// the returned 64bit integer. If count bits cannot be read then io.ErrUnexpectedEOF
// is returned.
func (r *Reader) Read(count int) (uint64, error) {
	if r.err != nil {
		return 0, r.err
	}

	// Requested count can be covered by the bit buffer alone, so
	// take count bits from the higher positions of the bit buffer.
	if count <= r.count {
		value := r.buf >> (r.count - count)

		r.count -= count
		r.buf &= (1 << r.count) - 1

		return value, nil
	}

	// Start off with whatever is in the bit buffer.
	value := r.buf
	count -= r.count

	data := make([]byte, 8)
	n, err := r.br.Read(data)
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	if err != nil {
		r.err = err
		return 0, r.err
	}

	r.count = n * 8
	r.buf = bufJoin(data[0:n])
	if count > r.count {
		r.err = io.ErrUnexpectedEOF
		return 0, r.err
	}

	// Fill in the rest of the value with the new data in the
	// higher positions of the bit buffer
	value = (value << count) | (r.buf >> (r.count - count))

	r.count -= count
	r.buf &= (1 << r.count) - 1

	return value, nil
}

// bufJoin joins the given byte segments into the lower positions of a 64bit
// integer. The given byte segements must not be more than 8 bytes.
func bufJoin(data []byte) uint64 {
	var value uint64

	for _, b := range data {
		value = (value << 8) | uint64(b)
	}

	return value
}
