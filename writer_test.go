package buffbits

import (
	"bytes"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestNewWriter(t *testing.T) {
	t.Parallel()

	writer := NewWriter(ioutil.Discard)

	assert.Equal(t, writer.bw.Size(), writer.bw.Available())
	assert.Equal(t, uint64(0), writer.buf)
	assert.Equal(t, 0, writer.count)
	assert.NoError(t, writer.err)
}

func TestNewWriterSize(t *testing.T) {
	t.Parallel()

	size := 2048
	writer := NewWriterSize(ioutil.Discard, size)

	assert.Equal(t, size, writer.bw.Size())
	assert.Equal(t, size, writer.bw.Available())
	assert.Equal(t, uint64(0), writer.buf)
	assert.Equal(t, 0, writer.count)
	assert.NoError(t, writer.err)
}

func TestWriterErr(t *testing.T) {
	size := 7

	t.Run("no error has occurred", func(t *testing.T) {
		t.Parallel()

		writer := NewWriterSize(ioutil.Discard, size)

		writer.Write(0, (size+1)*8) // Write enough bits to trigger the buffer to flush
		assert.NoError(t, writer.Err())
	})

	t.Run("error occurred during Write", func(t *testing.T) {
		t.Parallel()

		writer := NewWriterSize(errReadWriter{err: io.ErrNoProgress}, size)

		writer.Write(0, (size+1)*8) // Write enough bits to trigger the buffer to flush
		assert.ErrorIs(t, writer.Err(), io.ErrNoProgress)
	})

	t.Run("error occurred during Flush", func(t *testing.T) {
		t.Parallel()

		writer := NewWriterSize(errReadWriter{err: io.ErrNoProgress}, size)

		writer.Write(0, 8) // Write enough to allow a flush
		assert.NoError(t, writer.Err())

		writer.Flush()
		assert.ErrorIs(t, writer.Err(), io.ErrNoProgress)
	})
}

func TestWriterReset(t *testing.T) {
	writer := NewWriter(ioutil.Discard)
	writer.Write(0, 64)
	writer.Write(0, 16)
	assert.NoError(t, writer.Err())
	writer.err = io.ErrNoProgress

	var out bytes.Buffer
	writer.Reset(&out)

	writer.Write(0, 64)
	writer.Write(0, 16)
	writer.Flush()
	assert.NoError(t, writer.Err())

	assert.Equal(t, make([]byte, 10), out.Bytes())
}

func TestWriterWrite(t *testing.T) {
	t.Run("writer has errored previously", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)
		writer.err = io.ErrNoProgress

		err := writer.Write(0, 0)
		assert.ErrorIs(t, err, io.ErrNoProgress)
	})

	t.Run("write with more than max bits", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)

		err := writer.Write(0, 65)
		assert.ErrorIs(t, err, ErrInvalidCount)
	})

	t.Run("write with less than 0 bits", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)

		err := writer.Write(0, -1)
		assert.ErrorIs(t, err, ErrInvalidCount)
	})

	t.Run("bits that have unclean bit positions set", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)

		writer.Write(0b10010, 4)
		writer.Write(0b100, 2)

		assert.NoError(t, writer.Err())
		assert.Equal(t, 6, writer.count)
		assert.Equal(t, uint64(0b001000), writer.buf)
	})

	t.Run("bits within bit buffer limit", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)

		writer.Write(0b101, 3)
		writer.Write(0b110001, 6)
		writer.Write(0b0001, 4)
		writer.Write(0b111, 3)

		assert.NoError(t, writer.Err())
		assert.Equal(t, 16, writer.count)
		assert.Equal(t, uint64(0b1011100010001111), writer.buf)
	})

	t.Run("bits that reach the bit buffer limit", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		writer := NewWriter(&out)

		writer.Write(0b00000001, 8)
		writer.Write(0b00000011, 8)
		writer.Write(0b0000011100001111, 16)
		writer.Write(0b00011111001111110111111111111111, 32)

		assert.NoError(t, writer.Err())
		assert.Equal(t, 0, writer.count)
		assert.Equal(t, uint64(0), writer.buf)

		err := writer.bw.Flush()
		assert.NoError(t, err)

		assert.Equal(t, 8, out.Len())
		assert.Equal(t, []byte{
			0b00000001,
			0b00000011,
			0b00000111,
			0b00001111,
			0b00011111,
			0b00111111,
			0b01111111,
			0b11111111,
		}, out.Bytes())
	})

	t.Run("bits that exceed the bit buffer limit", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		writer := NewWriter(&out)

		writer.Write(0b0000000100000011, 16)
		writer.Write(0b0000000100000011000001110000111100011111001111110111111111111111, 64)
		assert.NoError(t, writer.Err())

		assert.Equal(t, 16, writer.count)
		assert.Equal(t, uint64(0b0111111111111111), writer.buf)

		err := writer.bw.Flush()
		assert.NoError(t, err)

		assert.Equal(t, 8, out.Len())
		assert.Equal(t, []byte{
			0b00000001,
			0b00000011,
			0b00000001,
			0b00000011,
			0b00000111,
			0b00001111,
			0b00011111,
			0b00111111,
		}, out.Bytes())
	})
}

func TestWriterFlush(t *testing.T) {
	t.Run("writer has errored previously", func(t *testing.T) {
		t.Parallel()

		writer := NewWriter(ioutil.Discard)
		writer.err = io.ErrNoProgress

		err := writer.Flush()
		assert.ErrorIs(t, io.ErrNoProgress, err)
	})

	t.Run("with no data in bit buffer", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		writer := NewWriter(&out)

		writer.Flush()
		assert.NoError(t, writer.Err())

		assert.Equal(t, 0, writer.count)
		assert.Equal(t, uint64(0), writer.buf)
		assert.Equal(t, 0, out.Len())
	})

	t.Run("with byte aligned data in bit buffer", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		writer := NewWriter(&out)

		writer.Write(0b0000000100000011, 16)
		writer.Flush()
		assert.NoError(t, writer.Err())

		assert.Equal(t, 0, writer.count)
		assert.Equal(t, uint64(0), writer.buf)
		assert.Equal(t, 2, out.Len())
		assert.Equal(t, []byte{
			0b00000001,
			0b00000011,
		}, out.Bytes())
	})

	t.Run("with byte misaligned data in bit buffer", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		writer := NewWriter(&out)

		writer.Write(0b000000010011, 12)
		writer.Flush()
		assert.NoError(t, writer.Err())

		assert.Equal(t, 0, writer.count)
		assert.Equal(t, uint64(0), writer.buf)
		assert.Equal(t, 2, out.Len())
		assert.Equal(t, []byte{
			0b00000001,
			0b00110000,
		}, out.Bytes())
	})
}

func TestFuzzWriter(t *testing.T) {
	t.Parallel()

	seed := time.Now().UnixNano()
	source := rand.NewSource(seed)
	rng := rand.New(source)
	fuzzer := fuzz.New().RandSource(source).NilChance(0).NumElements(1, 4096)

	var out bytes.Buffer
	writer := NewWriter(&out)

	// Generate a random number of raw bits.
	var raw []bool
	fuzzer.Fuzz(&raw)

	// Get the expected output from the raw bits and include any padding that would be written.
	var count int
	var j int
	bytesCount := (len(raw) + 8 - 1) / 8
	padding := bytesCount*8 - len(raw)
	expected := make([]byte, bytesCount)
	for i, b := range raw {
		var v byte
		if b {
			v = 1
		}

		expected[j] = (expected[j] << 1) | v
		count++

		if count == 8 {
			j++
			count = 0
		} else if i+1 == len(raw) {
			expected[j] <<= padding
		}
	}

	// Write the raw bits in random bit count intervals.
	for len(raw) != 0 {
		maxBits := 64
		if len(raw) < maxBits {
			maxBits = len(raw)
		}

		bitsCount := rng.Intn(maxBits) + 1
		data := raw[0:bitsCount]
		raw = raw[bitsCount:]

		var value uint64
		for _, b := range data {
			var v uint64
			if b {
				v = 1
			}

			value = (value << 1) | v
		}

		writer.Write(value, bitsCount)
	}

	writer.Flush()
	assert.NoError(t, writer.Err())

	if !assert.Equal(t, expected, out.Bytes()) {
		t.Logf("retry with seed = %#v", seed)
	}
}
