package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Client represents an S3 client for object storage operations
type Client struct {
	config   *Config
	s3Client *s3.Client
	uploader *manager.Uploader
}

// NewClient creates a new S3 client with the given configuration
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	clientConfig := *cfg
	clientConfig.Endpoint = strings.TrimSpace(clientConfig.Endpoint)
	clientConfig.PublicEndpoint = strings.TrimSpace(clientConfig.PublicEndpoint)
	clientConfig.Bucket = strings.TrimSpace(clientConfig.Bucket)

	if clientConfig.Bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if clientConfig.Endpoint == "" {
		return nil, fmt.Errorf("endpoint is required")
	}

	if clientConfig.AccessKeyID == "" {
		return nil, fmt.Errorf("access key ID is required")
	}

	if clientConfig.SecretAccessKey == "" {
		return nil, fmt.Errorf("secret access key is required")
	}

	if clientConfig.Region == "" {
		clientConfig.Region = "us-east-1" // default region
	}

	if clientConfig.AddressingStyle == "" {
		clientConfig.AddressingStyle = AddressingStylePath
	}
	if !clientConfig.AddressingStyle.isValid() {
		return nil, fmt.Errorf("invalid addressing style: %s", clientConfig.AddressingStyle)
	}

	if clientConfig.ObjectURLStyle == "" {
		if clientConfig.PublicEndpoint != "" {
			clientConfig.ObjectURLStyle = ObjectURLStylePublicEndpoint
		} else if clientConfig.AddressingStyle == AddressingStyleVirtualHost {
			clientConfig.ObjectURLStyle = ObjectURLStyleVirtualHost
		} else {
			clientConfig.ObjectURLStyle = ObjectURLStylePath
		}
	}
	if !clientConfig.ObjectURLStyle.isValid() {
		return nil, fmt.Errorf("invalid object URL style: %s", clientConfig.ObjectURLStyle)
	}

	endpoint, err := normalizeEndpoint(clientConfig.Endpoint, clientConfig.UseSSL)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}
	clientConfig.Endpoint = endpoint

	if clientConfig.PublicEndpoint != "" {
		publicEndpoint, err := normalizeEndpoint(clientConfig.PublicEndpoint, clientConfig.UseSSL)
		if err != nil {
			return nil, fmt.Errorf("invalid public endpoint URL: %w", err)
		}
		clientConfig.PublicEndpoint = publicEndpoint
	}

	// Create AWS configuration
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(clientConfig.Region),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     clientConfig.AccessKeyID,
				SecretAccessKey: clientConfig.SecretAccessKey,
			}, nil
		})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with custom endpoint
	s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(clientConfig.Endpoint)
		o.UsePathStyle = clientConfig.AddressingStyle != AddressingStyleVirtualHost
	})

	client := &Client{
		config:   &clientConfig,
		s3Client: s3Client,
		uploader: manager.NewUploader(s3Client),
	}

	return client, nil
}

// Upload uploads an object to S3 and returns the upload result with URL
func (c *Client) Upload(ctx context.Context, key string, data []byte, contentType string, metadata Metadata) (*UploadResult, error) {
	if key == "" {
		return nil, fmt.Errorf("object key cannot be empty")
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("data cannot be empty")
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Convert metadata to map[string]string
	metadataMap := make(map[string]string)
	for k, v := range metadata {
		metadataMap[k] = v
	}

	// Create input for PutObject
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
		Metadata:    metadataMap,
	}

	// Upload object
	result, err := c.s3Client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	// Build the full URL
	objectURL := c.buildObjectURL(key)

	uploadResult := &UploadResult{
		Key:      key,
		URL:      objectURL,
		ETag:     aws.ToString(result.ETag),
		Location: objectURL,
		Size:     int64(len(data)),
	}

	return uploadResult, nil
}

// UploadFromReader uploads an object from an io.Reader to S3 and returns the upload result with URL
func (c *Client) UploadFromReader(ctx context.Context, key string, reader io.Reader, size int64, contentType string, metadata Metadata) (*UploadResult, error) {
	if key == "" {
		return nil, fmt.Errorf("object key cannot be empty")
	}

	if reader == nil {
		return nil, fmt.Errorf("reader cannot be nil")
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Convert metadata to map[string]string
	metadataMap := make(map[string]string)
	for k, v := range metadata {
		metadataMap[k] = v
	}

	// Create input for PutObject
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.config.Bucket),
		Key:         aws.String(key),
		Body:        reader,
		ContentType: aws.String(contentType),
		Metadata:    metadataMap,
	}

	// Upload object
	result, err := c.s3Client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to upload object: %w", err)
	}

	// Build the full URL
	objectURL := c.buildObjectURL(key)

	uploadResult := &UploadResult{
		Key:      key,
		URL:      objectURL,
		ETag:     aws.ToString(result.ETag),
		Location: objectURL,
		Size:     size,
	}

	return uploadResult, nil
}

// Delete deletes an object from S3 and returns the delete result
func (c *Client) Delete(ctx context.Context, key string) (*DeleteResult, error) {
	if key == "" {
		return nil, fmt.Errorf("object key cannot be empty")
	}

	// Create input for DeleteObject
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	}

	// Delete object
	_, err := c.s3Client.DeleteObject(ctx, input)
	if err != nil {
		return &DeleteResult{
			Key:     key,
			Success: false,
			Message: fmt.Sprintf("failed to delete object: %v", err),
		}, fmt.Errorf("failed to delete object: %w", err)
	}

	return &DeleteResult{
		Key:     key,
		Success: true,
		Message: "object deleted successfully",
	}, nil
}

// DeleteMultiple deletes multiple objects from S3 and returns the delete results
func (c *Client) DeleteMultiple(ctx context.Context, keys []string) ([]*DeleteResult, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("keys cannot be empty")
	}

	// Prepare object identifiers
	objectIds := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		objectIds[i] = types.ObjectIdentifier{
			Key: aws.String(key),
		}
	}

	// Create input for DeleteObjects
	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(c.config.Bucket),
		Delete: &types.Delete{
			Objects: objectIds,
		},
	}

	// Delete objects
	result, err := c.s3Client.DeleteObjects(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to delete objects: %w", err)
	}

	// Build results
	results := make([]*DeleteResult, len(keys))

	// Mark deleted objects as successful
	for _, deleted := range result.Deleted {
		key := aws.ToString(deleted.Key)
		for i, k := range keys {
			if k == key {
				results[i] = &DeleteResult{
					Key:     key,
					Success: true,
					Message: "object deleted successfully",
				}
				break
			}
		}
	}

	// Mark failed objects
	for _, failure := range result.Errors {
		key := aws.ToString(failure.Key)
		for i, k := range keys {
			if k == key {
				results[i] = &DeleteResult{
					Key:     key,
					Success: false,
					Message: fmt.Sprintf("failed to delete: %s", aws.ToString(failure.Message)),
				}
				break
			}
		}
	}

	// Fill in any missing results
	for i, key := range keys {
		if results[i] == nil {
			results[i] = &DeleteResult{
				Key:     key,
				Success: false,
				Message: "unknown status",
			}
		}
	}

	return results, nil
}

// Get retrieves an object from S3 and returns the get result with object data
func (c *Client) Get(ctx context.Context, key string) (*GetResult, []byte, error) {
	if key == "" {
		return nil, nil, fmt.Errorf("object key cannot be empty")
	}

	// Create input for GetObject
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	}

	// Get object
	result, err := c.s3Client.GetObject(ctx, input)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer result.Body.Close()

	// Read object data
	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read object data: %w", err)
	}

	// Build the full URL
	objectURL := c.buildObjectURL(key)

	// Convert metadata
	metadata := make(Metadata)
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	getResult := &GetResult{
		Key:          key,
		URL:          objectURL,
		ETag:         aws.ToString(result.ETag),
		LastModified: formatTime(aws.ToTime(result.LastModified)),
		ContentType:  aws.ToString(result.ContentType),
		Size:         int64(len(data)),
		Metadata:     metadata,
	}

	return getResult, data, nil
}

// GetObjectURL returns the URL for an object without fetching it
func (c *Client) GetObjectURL(key string) string {
	if key == "" {
		return ""
	}
	return c.buildObjectURL(key)
}

// GetObjectInfo retrieves metadata for an object without downloading its content
func (c *Client) GetObjectInfo(ctx context.Context, key string) (*GetResult, error) {
	if key == "" {
		return nil, fmt.Errorf("object key cannot be empty")
	}

	// Create input for HeadObject
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	}

	// Get object metadata
	result, err := c.s3Client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object info: %w", err)
	}

	// Build the full URL
	objectURL := c.buildObjectURL(key)

	// Convert metadata
	metadata := make(Metadata)
	if result.Metadata != nil {
		for k, v := range result.Metadata {
			metadata[k] = v
		}
	}

	getResult := &GetResult{
		Key:          key,
		URL:          objectURL,
		ETag:         aws.ToString(result.ETag),
		LastModified: formatTime(aws.ToTime(result.LastModified)),
		ContentType:  aws.ToString(result.ContentType),
		Size:         aws.ToInt64(result.ContentLength),
		Metadata:     metadata,
	}

	return getResult, nil
}

// ListObjects lists objects in the bucket with the given prefix
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]*ObjectInfo, error) {
	// Create input for ListObjectsV2
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.config.Bucket),
		Prefix: aws.String(prefix),
	}

	// List objects
	result, err := c.s3Client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	// Build object info list
	objects := make([]*ObjectInfo, len(result.Contents))
	for i, obj := range result.Contents {
		metadata := make(Metadata)

		objects[i] = &ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: formatTime(aws.ToTime(obj.LastModified)),
			ETag:         aws.ToString(obj.ETag),
			StorageClass: string(obj.StorageClass),
			Metadata:     metadata,
		}
	}

	return objects, nil
}

// Exists checks if an object exists in the bucket
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if key == "" {
		return false, fmt.Errorf("object key cannot be empty")
	}

	// Create input for HeadObject
	input := &s3.HeadObjectInput{
		Bucket: aws.String(c.config.Bucket),
		Key:    aws.String(key),
	}

	_, err := c.s3Client.HeadObject(ctx, input)
	if err != nil {
		// Check if error is "NotFound"
		if strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// buildObjectURL builds the full URL for an object
func (c *Client) buildObjectURL(key string) string {
	endpoint := c.config.Endpoint
	if c.config.PublicEndpoint != "" {
		endpoint = c.config.PublicEndpoint
	}

	switch c.config.ObjectURLStyle {
	case ObjectURLStylePublicEndpoint:
		return joinURL(endpoint, key)
	case ObjectURLStyleVirtualHost:
		virtualHostEndpoint, err := bucketVirtualHostEndpoint(endpoint, c.config.Bucket)
		if err != nil {
			return joinURL(endpoint, c.config.Bucket, key)
		}
		return joinURL(virtualHostEndpoint, key)
	default:
		return joinURL(endpoint, c.config.Bucket, key)
	}
}

// formatTime formats a time.Time to a string
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}

// GetConfig returns the client configuration
func (c *Client) GetConfig() *Config {
	return c.config
}

func (s AddressingStyle) isValid() bool {
	return s == AddressingStylePath || s == AddressingStyleVirtualHost
}

func (s ObjectURLStyle) isValid() bool {
	return s == ObjectURLStylePath || s == ObjectURLStyleVirtualHost || s == ObjectURLStylePublicEndpoint
}

func normalizeEndpoint(endpoint string, useSSL bool) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("endpoint is required")
	}

	if !strings.Contains(endpoint, "://") {
		scheme := "https"
		if !useSSL {
			scheme = "http"
		}
		endpoint = scheme + "://" + endpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q", parsed.Scheme)
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), nil
}

func bucketVirtualHostEndpoint(endpoint, bucket string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("host is required")
	}

	parsed.Host = bucket + "." + parsed.Host
	return parsed.String(), nil
}

func joinURL(base string, parts ...string) string {
	joined, err := url.JoinPath(base, parts...)
	if err != nil {
		return strings.TrimRight(base, "/") + "/" + strings.Join(parts, "/")
	}
	return joined
}
