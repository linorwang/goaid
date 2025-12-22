package httpclient_test

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/linorwang/goaid/httpclient"
)

func TestClient_Get(t *testing.T) {
	client := httpclient.New()
	
	resp, err := client.Get(context.Background(), "https://httpbin.org/get")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_Post(t *testing.T) {
	client := httpclient.New()
	
	resp, err := client.Post(
		context.Background(), 
		"https://httpbin.org/post",
		httpclient.WithBody([]byte(`{"test": "data"}`)),
		httpclient.WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestClient_Concurrent(t *testing.T) {
	client := httpclient.New(
		httpclient.WithMaxIdleConns(100),
		httpclient.WithMaxIdleConnsPerHost(50),
	)
	
	var wg sync.WaitGroup
	concurrentRequests := 20
	
	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			
			resp, err := client.Get(context.Background(), "https://httpbin.org/get")
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			defer resp.Body.Close()
			
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}
		}()
	}
	
	wg.Wait()
}

func BenchmarkClient_Get(b *testing.B) {
	client := httpclient.New()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(context.Background(), "https://httpbin.org/get")
			if err != nil {
				b.Errorf("Expected no error, got %v", err)
				continue
			}
			resp.Body.Close()
		}
	})
}

func BenchmarkClient_Post(b *testing.B) {
	client := httpclient.New()
	body := []byte(`{"benchmark": "test"}`)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Post(
				context.Background(), 
				"https://httpbin.org/post",
				httpclient.WithBody(body),
				httpclient.WithHeader("Content-Type", "application/json"),
			)
			if err != nil {
				b.Errorf("Expected no error, got %v", err)
				continue
			}
			resp.Body.Close()
		}
	})
}