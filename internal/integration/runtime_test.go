//go:build integration

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	runtimeResultResponse = "response"
	runtimeResultError    = "error"
	runtimeResultInit     = "init-error"
)

type runtimeResult struct {
	kind string
	body []byte
}

type runtimeStub struct {
	server *httptest.Server
	result chan runtimeResult

	mu        sync.Mutex
	served    bool
	requestID string
}

func newRuntimeStub(t *testing.T) *runtimeStub {
	t.Helper()

	stub := &runtimeStub{
		result:    make(chan runtimeResult, 1),
		requestID: "integration-test-request",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/2018-06-01/runtime/invocation/next", stub.handleNext)
	mux.HandleFunc("/2018-06-01/runtime/invocation/", stub.handleInvocation)
	mux.HandleFunc("/2018-06-01/runtime/init/error", stub.handleInitError)

	stub.server = httptest.NewServer(mux)
	t.Cleanup(stub.server.Close)
	return stub
}

func (s *runtimeStub) Host() string {
	return strings.TrimPrefix(s.server.URL, "http://")
}

func (s *runtimeStub) Wait(t *testing.T, timeout time.Duration) runtimeResult {
	t.Helper()

	select {
	case result := <-s.result:
		return result
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for Lambda Runtime API result")
		return runtimeResult{}
	}
}

func (s *runtimeStub) handleNext(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	alreadyServed := s.served
	s.served = true
	s.mu.Unlock()

	if alreadyServed {
		<-r.Context().Done()
		return
	}

	w.Header().Set("Lambda-Runtime-Aws-Request-Id", s.requestID)
	w.Header().Set("Lambda-Runtime-Deadline-Ms", fmt.Sprintf("%d", time.Now().Add(30*time.Second).UnixMilli()))
	w.Header().Set("Lambda-Runtime-Invoked-Function-Arn", "arn:aws:lambda:us-east-1:000000000000:function:integration")
	w.Header().Set("Lambda-Runtime-Trace-Id", "Root=integration")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("null"))
}

func (s *runtimeStub) handleInvocation(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/2018-06-01/runtime/invocation/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != s.requestID {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	switch parts[1] {
	case "response":
		s.publish(runtimeResult{kind: runtimeResultResponse, body: body})
		w.WriteHeader(http.StatusAccepted)
	case "error":
		s.publish(runtimeResult{kind: runtimeResultError, body: body})
		w.WriteHeader(http.StatusAccepted)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (s *runtimeStub) handleInitError(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.publish(runtimeResult{kind: runtimeResultInit, body: body})
	w.WriteHeader(http.StatusAccepted)
}

func (s *runtimeStub) publish(result runtimeResult) {
	select {
	case s.result <- result:
	default:
	}
}
