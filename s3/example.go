package s3

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// Example demonstrates the usage of the S3 client
func Example() {
	// Create S3 client configuration
	cfg := &Config{
		Endpoint:        "https://s3.amazonaws.com", // or your S3-compatible endpoint
		AccessKeyID:     "your-access-key-id",
		SecretAccessKey: "your-secret-access-key",
		Region:          "us-east-1",
		Bucket:          "your-bucket-name",
		UseSSL:          true,
	}

	// Create S3 client
	client, err := NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	ctx := context.Background()

	// Example 1: Upload a file with byte data
	fmt.Println("=== Example 1: Upload file with byte data ===")
	data := []byte("Hello, this is a test file!")
	metadata := Metadata{
		"author":      "John Doe",
		"description": "Test file for S3 demo",
	}

	uploadResult, err := client.Upload(ctx, "test/hello.txt", data, "text/plain", metadata)
	if err != nil {
		log.Fatalf("Failed to upload file: %v", err)
	}

	fmt.Printf("Upload successful!\n")
	fmt.Printf("  Key: %s\n", uploadResult.Key)
	fmt.Printf("  URL: %s\n", uploadResult.URL)
	fmt.Printf("  ETag: %s\n", uploadResult.ETag)
	fmt.Printf("  Size: %d bytes\n", uploadResult.Size)
	fmt.Println()

	// Example 2: Get object info
	fmt.Println("=== Example 2: Get object info ===")
	getInfo, err := client.GetObjectInfo(ctx, "test/hello.txt")
	if err != nil {
		log.Fatalf("Failed to get object info: %v", err)
	}

	fmt.Printf("Object info retrieved!\n")
	fmt.Printf("  Key: %s\n", getInfo.Key)
	fmt.Printf("  URL: %s\n", getInfo.URL)
	fmt.Printf("  Size: %d bytes\n", getInfo.Size)
	fmt.Printf("  ContentType: %s\n", getInfo.ContentType)
	fmt.Println("  Metadata:")
	for k, v := range getInfo.Metadata {
		fmt.Printf("    %s: %s\n", k, v)
	}
	fmt.Println()

	// Example 3: Download/get object
	fmt.Println("=== Example 3: Download object ===")
	getResult, downloadedData, err := client.Get(ctx, "test/hello.txt")
	if err != nil {
		log.Fatalf("Failed to get object: %v", err)
	}

	fmt.Printf("Object downloaded!\n")
	fmt.Printf("  Key: %s\n", getResult.Key)
	fmt.Printf("  Size: %d bytes\n", getResult.Size)
	fmt.Printf("  Content: %s\n", string(downloadedData))
	fmt.Println()

	// Example 4: Upload from file reader
	fmt.Println("=== Example 4: Upload from file reader ===")
	file, err := os.Open("local_file.txt")
	if err != nil {
		log.Printf("Failed to open local file: %v", err)
	} else {
		defer file.Close()

		fileInfo, _ := file.Stat()
		uploadResult2, err := client.UploadFromReader(ctx, "test/uploaded_file.txt", file, fileInfo.Size(), "text/plain", Metadata{})
		if err != nil {
			log.Printf("Failed to upload from reader: %v", err)
		} else {
			fmt.Printf("Upload from reader successful!\n")
			fmt.Printf("  Key: %s\n", uploadResult2.Key)
			fmt.Printf("  URL: %s\n", uploadResult2.URL)
		}
		fmt.Println()
	}

	// Example 5: Check if object exists
	fmt.Println("=== Example 5: Check if object exists ===")
	exists, err := client.Exists(ctx, "test/hello.txt")
	if err != nil {
		log.Fatalf("Failed to check existence: %v", err)
	}
	fmt.Printf("Object exists: %v\n", exists)
	fmt.Println()

	// Example 6: List objects
	fmt.Println("=== Example 6: List objects ===")
	objects, err := client.ListObjects(ctx, "test/")
	if err != nil {
		log.Fatalf("Failed to list objects: %v", err)
	}

	fmt.Printf("Found %d objects:\n", len(objects))
	for _, obj := range objects {
		fmt.Printf("  - %s (%d bytes)\n", obj.Key, obj.Size)
	}
	fmt.Println()

	// Example 7: Get object URL without downloading
	fmt.Println("=== Example 7: Get object URL ===")
	url := client.GetObjectURL("test/hello.txt")
	fmt.Printf("Object URL: %s\n", url)
	fmt.Println()

	// Example 8: Delete object
	fmt.Println("=== Example 8: Delete object ===")
	deleteResult, err := client.Delete(ctx, "test/hello.txt")
	if err != nil {
		log.Fatalf("Failed to delete object: %v", err)
	}

	fmt.Printf("Delete result:\n")
	fmt.Printf("  Key: %s\n", deleteResult.Key)
	fmt.Printf("  Success: %v\n", deleteResult.Success)
	fmt.Printf("  Message: %s\n", deleteResult.Message)
	fmt.Println()

	// Example 9: Delete multiple objects
	fmt.Println("=== Example 9: Delete multiple objects ===")
	keysToDelete := []string{"test/file1.txt", "test/file2.txt"}
	deleteResults, err := client.DeleteMultiple(ctx, keysToDelete)
	if err != nil {
		log.Printf("Failed to delete multiple objects: %v", err)
	} else {
		fmt.Printf("Delete results:\n")
		for _, result := range deleteResults {
			fmt.Printf("  Key: %s, Success: %v, Message: %s\n", result.Key, result.Success, result.Message)
		}
	}
	fmt.Println()

	// Example 10: Upload and store URL to database (simulation)
	fmt.Println("=== Example 10: Upload and store URL for database ===")
	imageData := []byte("fake image data")
	imageMetadata := Metadata{
		"uploaded-by": "user123",
		"category":    "profile",
	}

	imageResult, err := client.Upload(ctx, "images/profile/user123.jpg", imageData, "image/jpeg", imageMetadata)
	if err != nil {
		log.Fatalf("Failed to upload image: %v", err)
	}

	// Simulate storing in database
	dbRecord := map[string]interface{}{
		"user_id":    123,
		"image_key":  imageResult.Key,
		"image_url":  imageResult.URL, // This is what you store in your database
		"image_etag": imageResult.ETag,
		"size":       imageResult.Size,
		"created_at": time.Now(),
	}

	fmt.Printf("Database record to save:\n")
	fmt.Printf("  user_id: %v\n", dbRecord["user_id"])
	fmt.Printf("  image_key: %v\n", dbRecord["image_key"])
	fmt.Printf("  image_url: %v\n", dbRecord["image_url"]) // Store this URL in your database
	fmt.Printf("  image_etag: %v\n", dbRecord["image_etag"])
	fmt.Printf("  size: %v\n", dbRecord["size"])
	fmt.Println()
}

// ExampleWithMinIO demonstrates usage with MinIO (S3-compatible storage)
func ExampleWithMinIO() {
	// MinIO configuration
	cfg := &Config{
		Endpoint:        "http://localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Region:          "us-east-1",
		Bucket:          "my-bucket",
		UseSSL:          false,
	}

	client, err := NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	ctx := context.Background()

	// Upload a file
	data := []byte("Hello MinIO!")
	result, err := client.Upload(ctx, "test.txt", data, "text/plain", Metadata{})
	if err != nil {
		log.Fatalf("Failed to upload: %v", err)
	}

	fmt.Printf("Uploaded to MinIO!\n")
	fmt.Printf("URL: %s\n", result.URL)
}

// ExampleWithCustomEndpoint demonstrates usage with any S3-compatible service
func ExampleWithCustomEndpoint(endpoint, accessKey, secretKey, bucket string) {
	cfg := &Config{
		Endpoint:        endpoint,
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Region:          "us-east-1",
		Bucket:          bucket,
		UseSSL:          true,
	}

	client, err := NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	// Use the client for your operations...
	_ = client
	_ = ctx
}
