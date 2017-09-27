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
