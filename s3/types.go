package s3

// Config holds the S3 client configuration
type Config struct {
	Endpoint        string // S3 endpoint URL (e.g., "https://s3.amazonaws.com")
	AccessKeyID     string // Access key ID
	SecretAccessKey string // Secret access key
	Region          string // AWS region (e.g., "us-east-1")
	Bucket          string // Bucket name
	UseSSL          bool   // Whether to use SSL (default: true)
}

// UploadResult contains the result of an upload operation
type UploadResult struct {
	Key          string // Object key (path in bucket)
	URL          string // Full URL to access the object
	ETag         string // Entity tag of the uploaded object
	UploadID     string // Upload ID for multipart uploads (if applicable)
	Location     string // Full location URL
	LastModified string // Last modified timestamp
	Size         int64  // Size of the uploaded object
}

// DeleteResult contains the result of a delete operation
type DeleteResult struct {
	Key     string // Object key that was deleted
	Success bool   // Whether the deletion was successful
	Message string // Additional message or error description
}

// GetResult contains the result of a get object operation
type GetResult struct {
	Key          string   // Object key
	URL          string   // Full URL to access the object
	ETag         string   // Entity tag
	LastModified string   // Last modified timestamp
	ContentType  string   // Content type
	Size         int64    // Size of the object
	Metadata     Metadata // User-defined metadata
}

// Metadata holds user-defined metadata for an object
type Metadata map[string]string

// ObjectInfo contains basic information about an object
type ObjectInfo struct {
	Key          string   // Object key
	Size         int64    // Size in bytes
	LastModified string   // Last modified timestamp
	ETag         string   // Entity tag
	StorageClass string   // Storage class
	Metadata     Metadata // User-defined metadata
}
