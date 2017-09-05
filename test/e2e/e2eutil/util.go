package e2eutil

import (
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

const (
	// Ephemeral port ranges
	ephemeralPortLowerBound = 50000
	ephemeralPortUpperBound = 60000
	targetVaultPort         = "8200"
)

var (
	// Atomic counter used to assign increasing port numbers
	portCounter int64
)

func init() {
	portCounter = ephemeralPortLowerBound
}

// NextPortNumber returns the next port number in an atomically increasing sequence in the ephemeral port range
func NextPortNumber() string {
	// TODO: Watch out for upper limit
	return strconv.FormatInt(atomic.AddInt64(&portCounter, 1), 10)
}

// GetPortMapping returns the []string{local:target} port mapping required for portforwarding
func GetPortMapping(localPort string) []string {
	return []string{localPort + ":" + targetVaultPort}
}

// PodLabelForOperator returns a label of the form name=<name>
func PodLabelForOperator(name string) map[string]string {
	return map[string]string{"name": name}
}

// LogfWithTimestamp is used for formatted test logs with the timestamp appended
func LogfWithTimestamp(t *testing.T, format string, args ...interface{}) {
	t.Log(time.Now(), fmt.Sprintf(format, args...))
}
