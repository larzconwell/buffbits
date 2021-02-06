package buffbits

import (
	"bufio"
	"io"
)

const (
	maxBits = 64
)

// Writer implements buffered bit level write access to an underlying io.Writer.
// Buffered writing is powered by the bufio package. If an error occurs while writing
// to a Writer, no more data will be written and all subsequent calls will return an
// error. After all data has been written, the client should call Flush to guarantee
// all the data has been written to the underlying io.Writer.
type Writer struct {
	bw    *bufio.Writer
	buf   uint64
	count int
	err   error
}

// NewWriter creates a buffered bit writer writing to w.
func NewWriter(w io.Writer) *Writer {
	return &Writer{bw: bufio.NewWriter(w)}
}

// NewWriterSize creates a buffered bit writer writing to w using a buffer of size bytes.
func NewWriterSize(w io.Writer, size int) *Writer {
	return &Writer{bw: bufio.NewWriterSize(w, size)}
}

// Err returns the first error that was encountered by the Writer.
func (w *Writer) Err() error {
	return w.err
}

// Write writes the lowest count bits of value to the Writer.
func (w *Writer) Write(value uint64, count int) error {
	if w.err != nil {
		return w.err
	}

	value &= (1 << count) - 1 // Clear value of positions set higher than count
	total := w.count + count

	if total < maxBits {
		w.count = total
		w.buf = (w.buf << count) | value

		return nil
	}

	// Get the higher bits of value and add them to get the filled bit buffer.
	higherValue := value >> (total - maxBits)
	out := (w.buf << (maxBits - w.count)) | higherValue

	// Reset bit buffer with the lower bits of value that are left over.
	w.buf = value & ((1 << (total - maxBits)) - 1)
	w.count = total - maxBits

	_, w.err = w.bw.Write(bufSplit(out, maxBits))
	return w.err
}

// Flush writes all the buffered data and pads to the last byte with 0s.
func (w *Writer) Flush() error {
	if w.err != nil {
		return w.err
	}

	// Ensure we're byte aligned by filling up the incomplete byte with 0s.
	nearestByteAlign := (w.count + 8 - 1) / 8 * 8
	if w.count != nearestByteAlign {
		w.err = w.Write(0, nearestByteAlign-w.count)
		if w.err != nil {
			return w.err
		}
	}

	if w.count > 0 {
		_, w.err = w.bw.Write(bufSplit(w.buf, w.count))
		if w.err != nil {
			return w.err
		}

		w.count = 0
		w.buf = 0
	}

	w.err = w.bw.Flush()
	return w.err
}

// bufSplit splits the byte segments on the lower end of value up to count bits.
// count must be byte aligned and must not be more than 64.
func bufSplit(value uint64, count int) []byte {
	list := make([]byte, count/8)

	for i := range list {
		shift := count - (i+1)*8

		list[i] = byte(value >> shift)
	}

	return list
}
