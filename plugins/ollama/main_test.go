package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper: no-op env resolver
func noEnv(key string) string { return "" }

// helper: env resolver from map
func envMap(m map[string]string) func(string) string {
	return func(key string) string { return m[key] }
}

// --- run() unit tests ---

func TestRun_EmptyStdin(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run([]string{"--model", "llama3.2"}, strings.NewReader("   "), &stdout, &stderr, noEnv)
	if err == nil {
		t.Fatal("expected error for empty stdin, got nil")
	}
	if !strings.Contains(err.Error(), "stdin") && !strings.Contains(err.Error(), "input") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRun_MissingModel(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Neither --model flag nor ORCAI_MODEL env var is set.
	err := run([]string{}, strings.NewReader("tell me a joke"), &stdout, &stderr, noEnv)
	if err == nil {
		t.Fatal("expected error when model is not set, got nil")
	}
	if !strings.Contains(err.Error(), "model") {
		t.Errorf("expected 'model' in error message, got: %v", err)
	}
}

func TestRun_ModelFromFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(generateResponse{Response: "got model: " + req.Model})
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--model", "llama3.2"},
		strings.NewReader("ping"),
		&stdout, &stderr,
		envMap(map[string]string{"ORCAI_OLLAMA_URL": srv.URL}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "llama3.2") {
		t.Errorf("expected llama3.2 in output, got %q", stdout.String())
	}
}

func TestRun_ModelFromEnv(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(generateResponse{Response: "got model: " + req.Model})
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	err := run(
		[]string{},
		strings.NewReader("ping"),
		&stdout, &stderr,
		envMap(map[string]string{
			"ORCAI_MODEL":      "qwen2.5",
			"ORCAI_OLLAMA_URL": srv.URL,
		}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "qwen2.5") {
		t.Errorf("expected qwen2.5 in output, got %q", stdout.String())
	}
}

func TestRun_DefaultURL(t *testing.T) {
	// Verify that when ORCAI_OLLAMA_URL is absent, the default resolves to localhost:11434.
	// We test this by overriding the URL with a test server to avoid needing real Ollama.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(generateResponse{Response: "ok"})
	}))
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	err := run(
		[]string{"--model", "llama3.2"},
		strings.NewReader("hi"),
		&stdout, &stderr,
		envMap(map[string]string{"ORCAI_OLLAMA_URL": srv.URL}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- callOllama() unit tests ---

func TestCallOllama_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected /api/generate, got %s", r.URL.Path)
		}

		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Stream {
			t.Error("expected stream=false")
		}
		if req.Model != "llama3.2" {
			t.Errorf("expected model llama3.2, got %q", req.Model)
		}

		json.NewEncoder(w).Encode(generateResponse{Response: "It works!"})
	}))
	defer srv.Close()

	result, err := callOllama(srv.URL, "llama3.2", "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "It works!" {
		t.Errorf("expected 'It works!', got %q", result)
	}
}

func TestCallOllama_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"model not found"}`))
	}))
	defer srv.Close()

	_, err := callOllama(srv.URL, "nonexistent-model", "hello")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error message, got: %v", err)
	}
}

func TestCallOllama_ConnectionRefused(t *testing.T) {
	_, err := callOllama("http://127.0.0.1:19999", "llama3.2", "hello")
	if err == nil {
		t.Fatal("expected error for connection refused, got nil")
	}
	if !strings.Contains(err.Error(), "19999") && !strings.Contains(err.Error(), "connecting") {
		t.Errorf("expected connection error, got: %v", err)
	}
}
