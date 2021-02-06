package buffbits

import (
	"bytes"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestNewReader(t *testing.T) {
	t.Parallel()

	reader := NewReader(strings.NewReader(""))

	assert.Greater(t, reader.br.Size(), 0)
	assert.Equal(t, uint64(0), reader.buf)
	assert.Equal(t, 0, reader.count)
	assert.NoError(t, reader.err)
}

func TestNewReaderSize(t *testing.T) {
	t.Parallel()

	size := 2048
	reader := NewReaderSize(strings.NewReader(""), size)

	assert.Equal(t, size, reader.br.Size())
	assert.Equal(t, uint64(0), reader.buf)
	assert.Equal(t, 0, reader.count)
	assert.NoError(t, reader.err)
}

func TestReaderErr(t *testing.T) {
	t.Run("no error has occurred", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{0}))

		reader.Read(8)
		assert.NoError(t, reader.Err())
	})

	t.Run("error occurred during Read", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(errReadWriter{err: io.ErrNoProgress})

		reader.Read(8)
		assert.ErrorIs(t, io.ErrNoProgress, reader.Err())
	})
}

func TestReaderRead(t *testing.T) {
	t.Run("reader has errored previously", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(strings.NewReader(""))
		reader.err = io.ErrNoProgress

		_, err := reader.Read(1)
		assert.ErrorIs(t, io.ErrNoProgress, err)
	})

	t.Run("underlying reader returns an error", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(errReadWriter{err: io.ErrNoProgress})

		_, err := reader.Read(1)
		assert.ErrorIs(t, io.ErrNoProgress, err)
	})

	t.Run("more bits than are available in the underlying reader", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{0}))

		_, err := reader.Read(9)
		assert.ErrorIs(t, io.ErrUnexpectedEOF, err)
	})

	t.Run("underlying reader reached EOF when reading bits", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{0}))

		_, err := reader.Read(8)
		assert.NoError(t, err)

		_, err = reader.Read(1)
		assert.ErrorIs(t, io.ErrUnexpectedEOF, err)
	})

	t.Run("bits within the bit buffer limit", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{
			0b10010100,
			0b11010001,
		}))

		reader.Read(1) // Trigger a read from the underlying reader
		value, err := reader.Read(12)
		assert.NoError(t, err)
		assert.Equal(t, uint64(0b001010011010), value)
	})

	t.Run("bits that reach the bit buffer limit", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{
			0b00000001,
			0b00000011,
			0b00000111,
			0b00001111,
			0b00011111,
			0b00111111,
			0b01111111,
			0b11111111,
		}))

		reader.Read(1) // Trigger a read from the underlying reader
		value, err := reader.Read(63)
		assert.NoError(t, err)
		assert.Equal(t, uint64(0b0000000100000011000001110000111100011111001111110111111111111111), value)
	})

	t.Run("bits that exceed bit buffer limit", func(t *testing.T) {
		t.Parallel()

		reader := NewReader(bytes.NewReader([]byte{
			0b00000001,
			0b00000011,
			0b00000111,
			0b00001111,
			0b00011111,
			0b00111111,
			0b01111111,
			0b11111111,
			0b00000001,
			0b00000011,
		}))

		reader.Read(16) // Trigger a read from the underlying reader and read a bunch from the buffer
		value, err := reader.Read(64)
		assert.NoError(t, err)
		assert.Equal(t, uint64(0b0000011100001111000111110011111101111111111111110000000100000011), value)
	})
}

func TestFuzzReader(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	rng := rand.New(source)
	fuzzer := fuzz.New().RandSource(source).NilChance(0).NumElements(1, 4096)

	var data []byte
	fuzzer.Fuzz(&data)

	reader := NewReader(bytes.NewReader(data))

	// Read the data stream in intervals of random bit counts.
	var out bytes.Buffer
	writer := NewWriter(&out)
	left := len(data) * 8
	for left != 0 {
		maxBits := 64
		if left < maxBits {
			maxBits = left
		}

		bitsCount := rng.Intn(maxBits) + 1
		left -= bitsCount

		value, err := reader.Read(bitsCount)
		assert.NoError(t, err)

		writer.Write(value, bitsCount)
	}

	writer.Flush()
	assert.NoError(t, writer.Err())

	if !assert.Equal(t, data, out.Bytes()) {
		t.Logf("retry with seed = %#v", seed)
	}
}
