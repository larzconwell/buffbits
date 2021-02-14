package buffbits

import (
	"flag"
)

var fuzzRetrySeed = flag.Int64("seed", 0, "seed to use running fuzz tests, use in conjunction with '-run TestFuzzWriter' or '-run TestFuzzReader'")

type errReadWriter struct {
	err error
}

func (erw errReadWriter) Write(b []byte) (int, error) {
	return len(b), erw.err
}

func (erw errReadWriter) Read(p []byte) (int, error) {
	return 0, erw.err
}
