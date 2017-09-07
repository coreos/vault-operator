package e2eutil

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type SampleSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// MapSecretToArbitraryData converts the obj(custom data type) to the arbitrary data format (map[string]interface{})
// that is used as the data in a vault secret. See https://github.com/hashicorp/vault/blob/master/api/secret.go#L19-L21
func MapObjectToArbitraryData(obj interface{}) (map[string]interface{}, error) {
	var data map[string]interface{}
	byt, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(byt, &data)
	return data, err
}

// PodLabelForOperator returns a label of the form name=<name>
func PodLabelForOperator(name string) map[string]string {
	return map[string]string{"name": name}
}

// LogfWithTimestamp is used for formatted test logs with the timestamp appended
func LogfWithTimestamp(t *testing.T, format string, args ...interface{}) {
	t.Log(time.Now(), fmt.Sprintf(format, args...))
}
