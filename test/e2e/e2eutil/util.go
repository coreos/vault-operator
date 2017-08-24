package e2eutil

import (
	"fmt"
	"testing"
	"time"
)

// PodLabelForOperator returns a label of the form name=<name>
func PodLabelForOperator(name string) map[string]string {
	return map[string]string{"name": name}
}

// LogfWithTimestamp is used for formatted test logs with the timestamp appended
func LogfWithTimestamp(t *testing.T, format string, args ...interface{}) {
	t.Log(time.Now(), fmt.Sprintf(format, args...))
}
