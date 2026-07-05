package httpclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func ExampleClient_Do() {
	client := New()

	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.org/get",
		WithQueryParam("page", "1"),
	)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	if !resp.Success() {
		log.Fatalf("HTTP error: %v", resp.Error())
	}

	fmt.Printf("status: %d\n", resp.StatusCode)
}

func ExampleClient_Do_withJSON() {
	client := New()

	payload := map[string]string{
		"name": "Ada",
	}
	resp, err := client.Do(
		context.Background(),
		http.MethodPost,
		"https://httpbin.org/post",
		WithJSON(payload),
		WithBearerToken("token"),
	)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	if !resp.Success() {
		log.Fatalf("HTTP error: %v", resp.Error())
	}

	fmt.Printf("status: %d\n", resp.StatusCode)
}

func ExampleClient_Get() {
	client := New()

	resp, err := client.Get(context.Background(), "https://httpbin.org/get")
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	body, err := ReadAllResponseBody(resp)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}

	fmt.Printf("status: %d, body bytes: %d\n", resp.StatusCode, len(body))
}

func ExampleClient_Use() {
	client := New(
		WithClientTimeout(30*time.Second),
		WithDefaultMaxRetries(2),
	)
	client.Use(
		NewLoggerMiddleware(nil),
		NewRequestIDMiddleware(nil),
		NewUserAgentMiddleware("GoAid-HttpClient/1.0"),
	)

	resp, err := client.Do(context.Background(), http.MethodGet, "https://httpbin.org/get")
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}

	fmt.Printf("status: %d\n", resp.StatusCode)
}
