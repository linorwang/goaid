package qrcode_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/linorwang/goaid/qrcode"
	"github.com/makiuchi-d/gozxing"
	zxingqrcode "github.com/makiuchi-d/gozxing/qrcode"
)

func TestGeneratePNGRoundTrip(t *testing.T) {
	t.Parallel()
	contents := []string{
		"https://example.com/orders/1001",
		"企业二维码工具包",
		`{"id":1001,"active":true}`,
		"emoji: 🧭✅",
	}
	for _, content := range contents {
		content := content
		t.Run(content, func(t *testing.T) {
			t.Parallel()
			data, err := qrcode.Generate(content)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
				t.Fatal("Generate() did not return a PNG")
			}
			if decoded := decodePNG(t, data); decoded != content {
				t.Fatalf("decoded content = %q, want %q", decoded, content)
			}
		})
	}
}

func TestGenerateSVG(t *testing.T) {
	t.Parallel()
	data, err := qrcode.Generate(
		"https://example.com/svg",
		qrcode.WithFormat(qrcode.FormatSVG),
		qrcode.WithSize(320),
		qrcode.WithForeground(color.NRGBA{R: 0x14, G: 0x45, B: 0x2f, A: 0xff}),
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	var document struct {
		XMLName xml.Name
		Width   int `xml:"width,attr"`
		Height  int `xml:"height,attr"`
	}
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatalf("SVG is not valid XML: %v\n%s", err, data)
	}
	if document.XMLName.Local != "svg" || document.Width != 320 || document.Height != 320 {
		t.Fatalf("unexpected SVG root: %+v", document)
	}
	if !bytes.Contains(data, []byte(`<path fill="#14452f"`)) {
		t.Fatal("SVG is missing the configured foreground path")
	}
}

func TestGenerateImageAndTransparency(t *testing.T) {
	t.Parallel()
	generated, err := qrcode.GenerateImage("transparent", qrcode.WithTransparent(true))
	if err != nil {
		t.Fatalf("GenerateImage() error = %v", err)
	}
	if generated.Bounds() != image.Rect(0, 0, qrcode.DefaultSize, qrcode.DefaultSize) {
		t.Fatalf("bounds = %v", generated.Bounds())
	}
	_, _, _, alpha := generated.At(0, 0).RGBA()
	if alpha != 0 {
		t.Fatalf("background alpha = %d, want 0", alpha)
	}

	_, err = qrcode.GenerateImage("svg", qrcode.WithFormat(qrcode.FormatSVG))
	if !errors.Is(err, qrcode.ErrImageUnavailable) {
		t.Fatalf("GenerateImage(SVG) error = %v, want ErrImageUnavailable", err)
	}
}

func TestLogoPNGRoundTripAndSVGEmbedding(t *testing.T) {
	t.Parallel()
	logo := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	draw.Draw(logo, logo.Bounds(), image.NewUniform(color.NRGBA{R: 0x16, G: 0x5d, B: 0xa8, A: 0xff}), image.Point{}, draw.Src)
	draw.Draw(logo, image.Rect(20, 20, 44, 44), image.NewUniform(color.White), image.Point{}, draw.Src)

	const content = "https://example.com/logo/enterprise-1001"
	data, err := qrcode.Generate(content, qrcode.WithLogoImage(logo), qrcode.WithLogoRatio(0.14))
	if err != nil {
		t.Fatalf("Generate() with logo error = %v", err)
	}
	if decoded := decodePNG(t, data); decoded != content {
		t.Fatalf("decoded content = %q, want %q", decoded, content)
	}

	var logoBuffer bytes.Buffer
	if err := png.Encode(&logoBuffer, logo); err != nil {
		t.Fatalf("encode test logo: %v", err)
	}
	svg, err := qrcode.Generate(
		content,
		qrcode.WithFormat(qrcode.FormatSVG),
		qrcode.WithLogoBytes(logoBuffer.Bytes()),
		qrcode.WithLogoRatio(0.14),
	)
	if err != nil {
		t.Fatalf("Generate(SVG) with logo error = %v", err)
	}
	if !bytes.Contains(svg, []byte(`href="data:image/png;base64,`)) {
		t.Fatal("SVG does not embed the logo as a data URI")
	}
}

func TestOutputHelpers(t *testing.T) {
	t.Parallel()
	var output bytes.Buffer
	if err := qrcode.Write(&output, "writer"); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if decoded := decodePNG(t, output.Bytes()); decoded != "writer" {
		t.Fatalf("decoded content = %q", decoded)
	}

	encoded, err := qrcode.GenerateBase64("base64")
	if err != nil {
		t.Fatalf("GenerateBase64() error = %v", err)
	}
	decodedBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("base64 decode error = %v", err)
	}
	if decoded := decodePNG(t, decodedBytes); decoded != "base64" {
		t.Fatalf("decoded content = %q", decoded)
	}

	dataURI, err := qrcode.GenerateDataURI("data-uri", qrcode.WithFormat(qrcode.FormatSVG))
	if err != nil {
		t.Fatalf("GenerateDataURI() error = %v", err)
	}
	if !strings.HasPrefix(dataURI, "data:image/svg+xml;base64,") {
		t.Fatalf("unexpected data URI prefix: %q", dataURI[:min(len(dataURI), 64)])
	}

	var typedNil *bytes.Buffer
	if err := qrcode.Write(typedNil, "nil writer"); !errors.Is(err, qrcode.ErrNilWriter) {
		t.Fatalf("Write(nil) error = %v, want ErrNilWriter", err)
	}
}

func TestSavePolicies(t *testing.T) {
	t.Parallel()
	filename := filepath.Join(t.TempDir(), "nested", "code.png")
	if err := qrcode.Save(filename, "first", qrcode.WithCreateParentDir(true)); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if decoded := decodePNG(t, data); decoded != "first" {
		t.Fatalf("decoded content = %q", decoded)
	}

	if err := qrcode.Save(filename, "second"); !errors.Is(err, qrcode.ErrFileExists) {
		t.Fatalf("second Save() error = %v, want ErrFileExists", err)
	}
	if err := qrcode.Save(filename, "second", qrcode.WithOverwrite(true)); err != nil {
		t.Fatalf("overwrite Save() error = %v", err)
	}
	data, err = os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile() after overwrite error = %v", err)
	}
	if decoded := decodePNG(t, data); decoded != "second" {
		t.Fatalf("decoded content after overwrite = %q", decoded)
	}

	temporaryFiles, err := filepath.Glob(filepath.Join(filepath.Dir(filename), ".qrcode-*.tmp"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(temporaryFiles) != 0 {
		t.Fatalf("temporary files were not cleaned: %v", temporaryFiles)
	}
}

func TestValidationErrors(t *testing.T) {
	t.Parallel()
	logo := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	tests := []struct {
		name    string
		content string
		opts    []qrcode.Option
		want    error
	}{
		{name: "empty content", want: qrcode.ErrEmptyContent},
		{name: "small size", content: "x", opts: []qrcode.Option{qrcode.WithSize(63)}, want: qrcode.ErrInvalidSize},
		{name: "negative margin", content: "x", opts: []qrcode.Option{qrcode.WithMargin(-1)}, want: qrcode.ErrInvalidMargin},
		{name: "format", content: "x", opts: []qrcode.Option{qrcode.WithFormat(qrcode.Format(99))}, want: qrcode.ErrUnsupportedFormat},
		{name: "correction", content: "x", opts: []qrcode.Option{qrcode.WithErrorCorrection(qrcode.ErrorCorrectionLevel(99))}, want: qrcode.ErrInvalidCorrection},
		{name: "same colors", content: "x", opts: []qrcode.Option{qrcode.WithForeground(color.White)}, want: qrcode.ErrInvalidColor},
		{name: "content safety limit", content: "long", opts: []qrcode.Option{qrcode.WithMaxContentBytes(3)}, want: qrcode.ErrContentTooLong},
		{name: "output limit", content: "x", opts: []qrcode.Option{qrcode.WithMaxOutputBytes(1)}, want: qrcode.ErrOutputTooLarge},
		{name: "bad logo bytes", content: "x", opts: []qrcode.Option{qrcode.WithLogoBytes([]byte("not an image"))}, want: qrcode.ErrInvalidLogo},
		{name: "logo dimensions", content: "x", opts: []qrcode.Option{qrcode.WithLogoImage(logo), qrcode.WithMaxLogoDimension(32)}, want: qrcode.ErrLogoTooLarge},
		{name: "logo plate ratio", content: "x", opts: []qrcode.Option{qrcode.WithLogoImage(logo), qrcode.WithLogoRatio(0.25)}, want: qrcode.ErrLogoTooLarge},
		{name: "nil option", content: "x", opts: []qrcode.Option{nil}, want: qrcode.ErrInvalidOption},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := qrcode.Generate(test.content, test.opts...)
			if !errors.Is(err, test.want) {
				t.Fatalf("Generate() error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := qrcode.GenerateContext(ctx, "cancelled")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("GenerateContext() error = %v, want context.Canceled", err)
	}
}

func TestGeneratorConcurrentUse(t *testing.T) {
	generator, err := qrcode.NewGenerator(qrcode.WithSize(256), qrcode.WithMargin(4))
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}
	const goroutines = 32
	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, goroutines)
	for index := 0; index < goroutines; index++ {
		index := index
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			data, err := generator.Generate("concurrent-" + string(rune('A'+index)))
			if err != nil {
				errorsChannel <- err
				return
			}
			if !bytes.HasPrefix(data, []byte("\x89PNG")) {
				errorsChannel <- errors.New("invalid PNG signature")
			}
		}()
	}
	waitGroup.Wait()
	close(errorsChannel)
	for err := range errorsChannel {
		t.Errorf("concurrent Generate() error = %v", err)
	}
}

func decodePNG(t *testing.T, data []byte) string {
	t.Helper()
	decodedImage, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}
	bitmap, err := gozxing.NewBinaryBitmapFromImage(decodedImage)
	if err != nil {
		t.Fatalf("NewBinaryBitmapFromImage() error = %v", err)
	}
	result, err := zxingqrcode.NewQRCodeReader().Decode(bitmap, nil)
	if err != nil {
		t.Fatalf("independent QR decode error = %v", err)
	}
	return result.GetText()
}
