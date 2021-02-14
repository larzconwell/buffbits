package buffbits

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestFuzzWriter(t *testing.T) {
	t.Parallel()

	iters := 1
	seed := *fuzzRetrySeed

	if seed == 0 {
		iters = 50
		seed = time.Now().UnixNano()
	}

	source := rand.NewSource(seed)
	rng := rand.New(source)
	fuzzer := fuzz.New().RandSource(source).NilChance(0).NumElements(1, 4096)

	for i := 0; i < iters; i++ {
		expected, actual, err := fuzzWriter(rng, fuzzer)
		assert.NoError(t, err)

		if !assert.Equal(t, expected, actual) {
			t.Logf("retry with '-seed %#v -run TestFuzzWriter'", seed)
		}
	}
}

func fuzzWriter(rng *rand.Rand, fuzzer *fuzz.Fuzzer) ([]byte, []byte, error) {
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
	return expected, out.Bytes(), writer.Err()
}
