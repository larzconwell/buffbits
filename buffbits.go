package buffbits

import (
	"errors"
)

const maxBits = 64

// ErrInvalidCount is the error used when a read/write occurs with a bit count that is too large or too small.
var ErrInvalidCount = errors.New("buffbits: read/write with invalid bit count")
