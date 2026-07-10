package qrcode

import (
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"io"
)

// Generator is an immutable, concurrency-safe QR Code generator.
type Generator struct {
	cfg options
}

func NewGenerator(opts ...Option) (*Generator, error) {
	cfg, err := applyOptions(opts)
	if err != nil {
		return nil, err
	}
	return &Generator{cfg: cfg}, nil
}

func Generate(content string, opts ...Option) ([]byte, error) {
	return GenerateContext(context.Background(), content, opts...)
}

func GenerateContext(ctx context.Context, content string, opts ...Option) ([]byte, error) {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return nil, err
	}
	return generator.GenerateContext(ctx, content)
}

func GenerateImage(content string, opts ...Option) (image.Image, error) {
	return GenerateImageContext(context.Background(), content, opts...)
}

func GenerateImageContext(ctx context.Context, content string, opts ...Option) (image.Image, error) {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return nil, err
	}
	return generator.GenerateImageContext(ctx, content)
}

func Write(w io.Writer, content string, opts ...Option) error {
	return WriteContext(context.Background(), w, content, opts...)
}

func WriteContext(ctx context.Context, w io.Writer, content string, opts ...Option) error {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return err
	}
	return generator.WriteContext(ctx, w, content)
}

func Save(filename, content string, opts ...Option) error {
	return SaveContext(context.Background(), filename, content, opts...)
}

func SaveContext(ctx context.Context, filename, content string, opts ...Option) error {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return err
	}
	return generator.SaveContext(ctx, filename, content)
}

func GenerateBase64(content string, opts ...Option) (string, error) {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return "", err
	}
	return generator.GenerateBase64(content)
}

func GenerateDataURI(content string, opts ...Option) (string, error) {
	generator, err := NewGenerator(opts...)
	if err != nil {
		return "", err
	}
	return generator.GenerateDataURI(content)
}

func (g *Generator) Generate(content string) ([]byte, error) {
	return g.GenerateContext(context.Background(), content)
}

func (g *Generator) GenerateContext(ctx context.Context, content string) ([]byte, error) {
	if g == nil {
		return nil, fmt.Errorf("%w: generator is nil", ErrInvalidOption)
	}
	if ctx == nil {
		return nil, fmt.Errorf("%w: context is nil", ErrInvalidOption)
	}
	matrix, err := encodeMatrix(checkedContext(ctx), content, g.cfg)
	if err != nil {
		return nil, err
	}

	switch g.cfg.format {
	case FormatPNG:
		image, err := renderImage(ctx, matrix, g.cfg)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return encodePNG(image, g.cfg.maxOutputBytes)
	case FormatSVG:
		return renderSVG(ctx, matrix, g.cfg)
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedFormat, g.cfg.format)
	}
}

func (g *Generator) GenerateImage(content string) (image.Image, error) {
	return g.GenerateImageContext(context.Background(), content)
}

func (g *Generator) GenerateImageContext(ctx context.Context, content string) (image.Image, error) {
	if g == nil {
		return nil, fmt.Errorf("%w: generator is nil", ErrInvalidOption)
	}
	if g.cfg.format != FormatPNG {
		return nil, fmt.Errorf("%w: %s", ErrImageUnavailable, g.cfg.format)
	}
	matrix, err := encodeMatrix(checkedContext(ctx), content, g.cfg)
	if err != nil {
		return nil, err
	}
	return renderImage(ctx, matrix, g.cfg)
}

func (g *Generator) Write(w io.Writer, content string) error {
	return g.WriteContext(context.Background(), w, content)
}

func (g *Generator) WriteContext(ctx context.Context, w io.Writer, content string) error {
	if isNil(w) {
		return ErrNilWriter
	}
	data, err := g.GenerateContext(ctx, content)
	if err != nil {
		return err
	}
	written, err := io.Copy(w, bytesReader(data))
	if err != nil {
		return fmt.Errorf("qrcode: write output: %w", err)
	}
	if written != int64(len(data)) {
		return io.ErrShortWrite
	}
	return nil
}

func (g *Generator) GenerateBase64(content string) (string, error) {
	data, err := g.Generate(content)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (g *Generator) GenerateDataURI(content string) (string, error) {
	encoded, err := g.GenerateBase64(content)
	if err != nil {
		return "", err
	}
	return "data:" + g.cfg.format.MIMEType() + ";base64," + encoded, nil
}

type readOnlyBytes struct {
	data   []byte
	offset int
}

func bytesReader(data []byte) *readOnlyBytes {
	return &readOnlyBytes{data: data}
}

func (r *readOnlyBytes) Read(destination []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(destination, r.data[r.offset:])
	r.offset += n
	return n, nil
}
