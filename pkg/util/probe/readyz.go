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

package probe

import (
	"net/http"
	"sync"
)

const (
	// HTTPReadyzEndpoint is the endpoint at which the readiness probe is supported
	HTTPReadyzEndpoint = "/readyz"
)

var (
	mu    sync.Mutex
	ready = false
)

// SetReady sets the ready condition to true, which causes the handler to respond with an OK status to readiness probes
func SetReady() {
	mu.Lock()
	ready = true
	mu.Unlock()
}

// ReadyzHandler writes back the HTTP status code 200 if the operator is ready, and 500 otherwise
func ReadyzHandler(w http.ResponseWriter, r *http.Request) {
	if isReady() {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

// isReady returns the status of the ready variable
func isReady() bool {
	mu.Lock()
	isReady := ready
	mu.Unlock()
	return isReady
}
