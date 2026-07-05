package captcha

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestImageCaptchaServiceSimpleAPI(t *testing.T) {
	ctx := context.Background()
	service := New(nil, CaptchaOption{
		IncludeValue: true,
	})

	resp, err := service.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.ID == "" {
		t.Fatal("expected captcha ID")
	}
	if resp.Value == "" {
		t.Fatal("expected captcha value when IncludeValue is enabled")
	}
	if !strings.HasPrefix(resp.ImageBase64, "data:image/png;base64,") {
		t.Fatalf("expected png data URI, got %q", resp.ImageBase64[:min(24, len(resp.ImageBase64))])
	}

	valid, err := service.Verify(ctx, resp.ID, resp.Value)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("expected captcha to be valid")
	}

	valid, err = service.Verify(ctx, resp.ID, resp.Value)
	if err != nil {
		t.Fatalf("Verify repeated captcha returned unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected captcha to be consumed after successful verification")
	}
}

func TestGenerateDoesNotExposeValueByDefault(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryCaptchaStore()
	service := New(store)

	resp, err := service.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if resp.Value != "" {
		t.Fatal("captcha value must not be exposed by default")
	}

	answer, err := store.Get(ctx, resp.ID)
	if err != nil {
		t.Fatalf("Get stored answer failed: %v", err)
	}

	valid, err := service.Verify(ctx, resp.ID, answer)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("expected stored answer to verify")
	}
}

func TestVerifyWrongAnswer(t *testing.T) {
	ctx := context.Background()
	service := New(nil, CaptchaOption{
		IncludeValue: true,
	})

	resp, err := service.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	valid, err := service.Verify(ctx, resp.ID, "wrong")
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected wrong answer to be invalid")
	}
}

func TestCaptchaExpiration(t *testing.T) {
	ctx := context.Background()
	service := New(nil, CaptchaOption{
		ExpireTime:   100 * time.Millisecond,
		IncludeValue: true,
	})

	resp, err := service.Generate(ctx)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	valid, err := service.Verify(ctx, resp.ID, resp.Value)
	if err != nil {
		t.Fatalf("Verify expired captcha returned unexpected error: %v", err)
	}
	if valid {
		t.Fatal("expected expired captcha to be invalid")
	}
}

func TestGenerateImageCaptchaWithCustomSize(t *testing.T) {
	ctx := context.Background()
	service := New(nil)

	resp, err := service.GenerateImageCaptcha(ctx, 200, 60)
	if err != nil {
		t.Fatalf("GenerateImageCaptcha failed: %v", err)
	}

	bounds := resp.Image.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 60 {
		t.Fatalf("expected 200x60 image, got %dx%d", bounds.Dx(), bounds.Dy())
	}
}
