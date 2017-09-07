package e2eutil

import (
	"sync/atomic"
)

const (
	// Ephemeral port ranges
	ephemeralPortLowerBound = 30000
	ephemeralPortUpperBound = 60000
)

var (
	// Atomic counter used to assign increasing port numbers
	portCounter int64
)

func init() {
	portCounter = ephemeralPortLowerBound
}

// NextPortNumber returns the next port number in an atomically increasing sequence in the ephemeral port range
func NextPortNumber() int {
	return int(atomic.AddInt64(&portCounter, 1))
}
