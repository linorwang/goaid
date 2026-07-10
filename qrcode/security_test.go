package qrcode_test

import (
	"context"
	"errors"
	"image"
	"image/color"
	"testing"

	"github.com/linorwang/goaid/qrcode"
)

type oversizedImage struct{}

func (oversizedImage) ColorModel() color.Model { return color.NRGBAModel }
func (oversizedImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, qrcode.DefaultMaxLogoDimension+1, 1)
}
func (oversizedImage) At(int, int) color.Color { return color.Black }

func TestLogoPreAllocationLimit(t *testing.T) {
	_, err := qrcode.NewGenerator(qrcode.WithLogoImage(oversizedImage{}))
	if !errors.Is(err, qrcode.ErrLogoTooLarge) {
		t.Fatalf("NewGenerator() error = %v, want ErrLogoTooLarge", err)
	}
}

func TestNilImageContext(t *testing.T) {
	_, err := qrcode.GenerateImageContext(nil, "nil context")
	if !errors.Is(err, qrcode.ErrInvalidOption) {
		t.Fatalf("GenerateImageContext(nil) error = %v, want ErrInvalidOption", err)
	}
}

func TestWiFiRejectsControlCharacters(t *testing.T) {
	_, err := qrcode.WiFi(qrcode.WiFiConfig{SSID: "office\nnetwork", Password: "secret", Encryption: qrcode.WPA})
	if !errors.Is(err, qrcode.ErrInvalidPayload) {
		t.Fatalf("WiFi() error = %v, want ErrInvalidPayload", err)
	}
}

var _ context.Context = context.Background()
