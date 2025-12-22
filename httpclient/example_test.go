package httpclient_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/linorwang/goaid/httpclient"
)

func ExampleNew() {
	// 创建一个默认的HTTP客户端
	client := httpclient.New()
	
	// 发送GET请求
	resp, err := client.Get(context.Background(), "https://httpbin.org/get")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %s\n", resp.Status)
	// Output: Status: 200 OK
}

func ExampleClient_Get() {
	client := httpclient.New(
		httpclient.WithClientTimeout(10*time.Second),
		httpclient.WithMaxIdleConns(50),
	)
	
	// 发送带查询参数的GET请求
	resp, err := client.Get(
		context.Background(), 
		"https://httpbin.org/get",
		httpclient.WithQueryParam("key", "value"),
		httpclient.WithHeader("User-Agent", "GoAid-HttpClient/1.0"),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %s\n", resp.Status)
}

func ExampleClient_Post() {
	client := httpclient.New()
	
	// 发送POST请求
	resp, err := client.Post(
		context.Background(),
		"https://httpbin.org/post",
		httpclient.WithBody([]byte(`{"name": "test", "value": 123}`)),
		httpclient.WithHeader("Content-Type", "application/json"),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	fmt.Printf("Status: %s\n", resp.Status)
}

func ExampleClient_Send() {
	client := httpclient.New()
	
	// 发送自定义方法请求
	resp, err := client.Send(
		context.Background(),
		http.MethodPut,
		"https://httpbin.org/put",
		httpclient.WithBody([]byte(`{"updated": true}`)),
		httpclient.WithHeader("Content-Type", "application/json"),
		httpclient.WithTimeout(5*time.Second),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := httpclient.ReadAllResponseBody(resp)
	fmt.Printf("Response: %s\n", string(body))
}