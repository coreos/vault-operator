// Copyright 2018 The vault-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
