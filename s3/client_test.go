package s3

import "testing"

func TestNewClientDefaultsForCompatibleS3(t *testing.T) {
	cfg := &Config{
		Endpoint:        "localhost:9000",
		AccessKeyID:     "minioadmin",
		SecretAccessKey: "minioadmin",
		Bucket:          "my-bucket",
		UseSSL:          false,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	got := client.GetConfig()
	if got.Endpoint != "http://localhost:9000" {
		t.Fatalf("Endpoint = %q, want %q", got.Endpoint, "http://localhost:9000")
	}
	if got.Region != "us-east-1" {
		t.Fatalf("Region = %q, want %q", got.Region, "us-east-1")
	}
	if got.AddressingStyle != AddressingStylePath {
		t.Fatalf("AddressingStyle = %q, want %q", got.AddressingStyle, AddressingStylePath)
	}
	if got.ObjectURLStyle != ObjectURLStylePath {
		t.Fatalf("ObjectURLStyle = %q, want %q", got.ObjectURLStyle, ObjectURLStylePath)
	}
}

func TestNewClientDefaultsToPublicEndpointURLs(t *testing.T) {
	cfg := &Config{
		Endpoint:        "http://localhost:9000",
		PublicEndpoint:  "https://cdn.example.com/assets",
		AccessKeyID:     "access-key",
		SecretAccessKey: "secret-key",
		Bucket:          "my-bucket",
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient returned error: %v", err)
	}

	got := client.GetObjectURL("images/a b.png")
	want := "https://cdn.example.com/assets/images/a%20b.png"
	if got != want {
		t.Fatalf("GetObjectURL() = %q, want %q", got, want)
	}
}

func TestBuildObjectURLStyles(t *testing.T) {
	tests := []struct {
		name string
		cfg  *Config
		want string
	}{
		{
			name: "path style",
			cfg: &Config{
				Endpoint:       "http://localhost:9000",
				Bucket:         "my-bucket",
				ObjectURLStyle: ObjectURLStylePath,
			},
			want: "http://localhost:9000/my-bucket/dir/a%20b.txt",
		},
		{
			name: "virtual host style",
			cfg: &Config{
				Endpoint:       "https://s3.amazonaws.com",
				Bucket:         "my-bucket",
				ObjectURLStyle: ObjectURLStyleVirtualHost,
			},
			want: "https://my-bucket.s3.amazonaws.com/dir/a%20b.txt",
		},
		{
			name: "public endpoint style",
			cfg: &Config{
				Endpoint:        "http://localhost:9000",
				PublicEndpoint:  "https://static.example.com",
				Bucket:          "my-bucket",
				ObjectURLStyle:  ObjectURLStylePublicEndpoint,
				AddressingStyle: AddressingStylePath,
			},
			want: "https://static.example.com/dir/a%20b.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{config: tt.cfg}
			got := client.buildObjectURL("dir/a b.txt")
			if got != tt.want {
				t.Fatalf("buildObjectURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
