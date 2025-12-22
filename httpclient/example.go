package httpclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ExampleUsage 演示如何使用HTTP客户端
func ExampleUsage() {
	// 创建一个带有自定义配置的客户端
	client := New(
		WithClientTimeout(30 * time.Second),
		WithMaxIdleConns(100),
		WithMaxIdleConnsPerHost(10),
	)

	// 发送GET请求
	resp, err := client.Get(
		context.Background(),
		"https://httpbin.org/get",
		WithQueryParam("key", "value"),
		WithHeader("User-Agent", "GoAid-HttpClient/1.0"),
	)
	if err != nil {
		log.Fatalf("Failed to send GET request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("GET Response Status: %s\n", resp.Status)

	// 发送POST请求
	resp, err = client.Post(
		context.Background(),
		"https://httpbin.org/post",
		WithBody([]byte(`{"name": "test", "value": 123}`)),
		WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		log.Fatalf("Failed to send POST request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("POST Response Status: %s\n", resp.Status)

	// 发送自定义方法请求
	resp, err = client.Send(
		context.Background(),
		http.MethodPut,
		"https://httpbin.org/put",
		WithBody([]byte(`{"updated": true}`)),
		WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		log.Fatalf("Failed to send PUT request: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("PUT Response Status: %s\n", resp.Status)
}