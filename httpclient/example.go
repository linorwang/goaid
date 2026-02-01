package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ExampleUsage 演示如何使用HTTP客户端（基础用法）
func ExampleUsage() {
	// 创建一个带有自定义配置的客户端
	client := New(
		WithClientTimeout(30*time.Second),
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

// ExampleEnhancedUsage 演示增强功能的使用
func ExampleEnhancedUsage() {
	// 创建带中间件的客户端
	client := New(
		WithClientTimeout(30*time.Second),
		WithDefaultMaxRetries(3),
		WithDefaultBackoffStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second)),
	)

	// 添加中间件
	client.Use(
		NewLoggerMiddleware(nil),    // 使用默认日志
		NewRequestIDMiddleware(nil), // 自动生成请求ID
		NewUserAgentMiddleware("GoAid-HttpClient/2.0"),
	)

	// 使用Do方法发送请求，返回包装后的响应
	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
		WithQueryParam("test", "value"),
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	// 使用增强的Response方法
	if resp.Success() {
		fmt.Printf("Request succeeded!\n")
		fmt.Printf("Status: %d\n", resp.StatusCode)
		fmt.Printf("Body: %s\n", resp.String())
	} else {
		// 处理错误
		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.IsNotFound() {
				fmt.Println("Resource not found")
			} else if httpErr.IsTimeout() {
				fmt.Println("Request timeout")
			} else if httpErr.IsServerError() {
				fmt.Println("Server error")
			}
		}
	}
}

// ExampleJSONHandling 演示JSON自动处理
func ExampleJSONHandling() {
	client := New()

	// 定义请求和响应结构
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	type Response struct {
		JSON User `json:"json"`
	}

	// 发送POST请求
	reqBody := User{
		Name:  "John Doe",
		Email: "john@example.com",
	}

	// 序列化请求体
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}

	// 发送请求
	resp, err := client.Do(
		context.Background(),
		http.MethodPost,
		"https://httpbin.com/post",
		WithBody(bodyBytes),
		WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	// 自动反序列化响应
	var result Response
	if err := resp.JSON(&result); err != nil {
		log.Fatalf("Failed to unmarshal response: %v", err)
	}

	fmt.Printf("Response: %+v\n", result.JSON)
}

// ExampleRetryMechanism 演示重试机制
func ExampleRetryMechanism() {
	client := New(
		WithDefaultMaxRetries(3), // 默认重试3次
		WithDefaultBackoffStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second)),
	)

	// 或者使用WithRetry为单个请求设置重试次数
	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
		WithRetry(5), // 这个请求重试5次
	)
	if err != nil {
		log.Fatalf("Request failed after retries: %v", err)
	}

	fmt.Printf("Request succeeded after retries: %d\n", resp.StatusCode)
}

// ExampleMiddlewareUsage 演示中间件使用
func ExampleMiddlewareUsage() {
	client := New()

	// 1. 日志中间件
	client.Use(
		NewLoggerMiddleware(nil),
	)

	// 2. 认证中间件
	client.Use(
		NewAuthMiddleware("your-api-token"),
	)

	// 3. 指标收集中间件
	var requestCount, successCount, errorCount int
	client.Use(
		NewMetricsMiddleware(
			func(method, url string) {
				requestCount++
				fmt.Printf("Request: %s %s\n", method, url)
			},
			func(method, url string, statusCode int, duration time.Duration) {
				successCount++
				fmt.Printf("Success: %s %s - Status: %d - Duration: %v\n",
					method, url, statusCode, duration)
			},
			func(method, url string, err error) {
				errorCount++
				fmt.Printf("Error: %s %s - Error: %v\n", method, url, err)
			},
		),
	)

	// 4. 自定义请求头中间件
	client.Use(
		NewHeaderMiddleware(map[string]string{
			"X-Custom-Header": "custom-value",
			"X-App-Version":   "1.0.0",
		}),
	)

	// 发送请求（会自动经过所有中间件）
	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("Response: %d\n", resp.StatusCode)
	fmt.Printf("Metrics - Total: %d, Success: %d, Error: %d\n",
		requestCount, successCount, errorCount)
}

// ExampleAdvancedConfiguration 演示高级配置
func ExampleAdvancedConfiguration() {
	client := New(
		// 连接池配置
		WithMaxIdleConns(200),
		WithMaxIdleConnsPerHost(100),
		WithMaxConnsPerHost(50),
		WithIdleConnTimeout(90*time.Second),
		WithKeepAlive(30*time.Second),

		// 超时配置
		WithClientTimeout(30*time.Second),
		WithResponseHeaderTimeout(10*time.Second),
		WithTLSHandshakeTimeout(10*time.Second),

		// HTTP/2配置
		WithForceAttemptHTTP2(true),

		// 重试配置
		WithDefaultMaxRetries(3),
		WithDefaultBackoffStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second)),
	)

	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	fmt.Printf("Response: %d\n", resp.StatusCode)
}

// ExampleClientClone 演示客户端克隆
func ExampleClientClone() {
	// 创建基础客户端
	baseClient := New(
		WithClientTimeout(30 * time.Second),
	)
	baseClient.Use(
		NewLoggerMiddleware(nil),
	)

	// 克隆客户端并添加额外的配置
	apiClient := baseClient.Clone()
	apiClient.Use(
		NewAuthMiddleware("api-token"),
	)

	// 克隆客户端用于不同的用途
	analyticsClient := baseClient.Clone()
	analyticsClient.Use(
		NewHeaderMiddleware(map[string]string{
			"X-Analytics-ID": "analytics-123",
		}),
	)

	// 使用不同的客户端
	resp1, err := apiClient.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
	)
	if err != nil {
		log.Fatalf("API request failed: %v", err)
	}
	fmt.Printf("API Response: %d\n", resp1.StatusCode)

	resp2, err := analyticsClient.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
	)
	if err != nil {
		log.Fatalf("Analytics request failed: %v", err)
	}
	fmt.Printf("Analytics Response: %d\n", resp2.StatusCode)
}

// ExampleErrorHandling 演示错误处理
func ExampleErrorHandling() {
	client := New()

	resp, err := client.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/status/404",
	)

	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}

	// 检查响应是否成功
	if !resp.Success() {
		if httpErr := resp.Error(); httpErr != nil {
			switch h := httpErr.(type) {
			case *HTTPError:
				if h.IsNotFound() {
					fmt.Println("资源不存在 (404)")
				} else if h.IsClientError() {
					fmt.Printf("客户端错误 (4xx): %d\n", h.StatusCode)
				} else if h.IsServerError() {
					fmt.Printf("服务器错误 (5xx): %d\n", h.StatusCode)
				} else if h.IsTimeout() {
					fmt.Println("请求超时")
				} else {
					fmt.Printf("HTTP错误: %d - %s\n", h.StatusCode, h.Message)
				}
			default:
				fmt.Printf("未知错误: %v\n", httpErr)
			}
		}
	}
}

// ExampleBackoffStrategies 演示不同的退避策略
func ExampleBackoffStrategies() {
	// 1. 指数退避（默认）
	client1 := New(
		WithDefaultMaxRetries(3),
		WithDefaultBackoffStrategy(NewExponentialBackoff(100*time.Millisecond, 5*time.Second)),
	)

	// 2. 线性退避
	_ = New(
		WithDefaultMaxRetries(3),
		WithDefaultBackoffStrategy(NewLinearBackoff(500*time.Millisecond)),
	)

	// 3. 常数退避
	_ = New(
		WithDefaultMaxRetries(3),
		WithDefaultBackoffStrategy(NewConstantBackoff(1*time.Second)),
	)

	// 使用任意客户端
	resp, err := client1.Do(
		context.Background(),
		http.MethodGet,
		"https://httpbin.com/get",
	)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	fmt.Printf("Response: %d\n", resp.StatusCode)
}
