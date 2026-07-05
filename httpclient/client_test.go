package httpclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetReturnsReadableBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	body, err := ReadAllResponseBody(resp)
	if err != nil {
		t.Fatalf("ReadAllResponseBody returned error: %v", err)
	}
	if string(body) != "ok" {
		t.Fatalf("body = %q, want %q", body, "ok")
	}
}

func TestDoReadsBodyAndDecodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":7,"name":"Ada"}`))
	}))
	defer server.Close()

	client := New()
	resp, err := client.Do(context.Background(), http.MethodGet, server.URL)
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if !resp.Success() {
		t.Fatalf("Success() = false, status = %d", resp.StatusCode)
	}

	var got struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := resp.JSON(&got); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}
	if got.ID != 7 || got.Name != "Ada" {
		t.Fatalf("decoded response = %+v", got)
	}
}

func TestWithJSONSetsBodyAndContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}

		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload.Name != "Ada" {
			t.Fatalf("payload.Name = %q, want Ada", payload.Name)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New()
	resp, err := client.Do(context.Background(), http.MethodPost, server.URL, WithJSON(map[string]string{"name": "Ada"}))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestRetryReplaysRequestBody(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if string(body) != "payload" {
			t.Fatalf("body = %q, want payload", body)
		}
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("done"))
	}))
	defer server.Close()

	client := New(WithDefaultBackoffStrategy(NewConstantBackoff(0)))
	resp, err := client.Do(context.Background(), http.MethodPost, server.URL, WithBody([]byte("payload")), WithRetry(1))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if resp.String() != "done" {
		t.Fatalf("response body = %q, want done", resp.String())
	}
}

func TestWithRetryZeroOverridesClientDefault(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New(
		WithDefaultMaxRetries(2),
		WithDefaultBackoffStrategy(NewConstantBackoff(0)),
	)
	resp, err := client.Do(context.Background(), http.MethodGet, server.URL, WithRetry(0))
	if err != nil {
		t.Fatalf("Do returned error: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}
