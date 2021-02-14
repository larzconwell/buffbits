package buffbits

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
)

func TestFuzzReader(t *testing.T) {
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
		expected, actual, err := fuzzReader(rng, fuzzer)
		assert.NoError(t, err)

		if !assert.Equal(t, expected, actual) {
			t.Logf("retry with '-seed %#v -run TestFuzzReader'", seed)
		}
	}
}

func fuzzReader(rng *rand.Rand, fuzzer *fuzz.Fuzzer) ([]byte, []byte, error) {
	var data []byte
	fuzzer.Fuzz(&data)

	reader := NewReader(bytes.NewReader(data))

	// Read the data stream in intervals of random bit counts.
	var actual bytes.Buffer
	writer := NewWriter(&actual)
	left := len(data) * 8
	for left != 0 {
		maxBits := 64
		if left < maxBits {
			maxBits = left
		}

		bitsCount := rng.Intn(maxBits) + 1
		left -= bitsCount

		value, err := reader.Read(bitsCount)
		if err != nil {
			return nil, nil, err
		}

		writer.Write(value, bitsCount)
	}

	writer.Flush()
	return data, actual.Bytes(), writer.Err()
}
