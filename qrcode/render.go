package qrcode

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strconv"
)

type renderLayout struct {
	scale       int
	moduleStart int
}

func calculateLayout(matrix [][]bool, cfg options) (renderLayout, error) {
	moduleCount := len(matrix)
	totalModules := moduleCount + 2*cfg.margin
	if totalModules <= 0 || cfg.size < totalModules {
		return renderLayout{}, fmt.Errorf("%w: size=%d requires_at_least=%d", ErrInvalidSize, cfg.size, totalModules)
	}
	scale := cfg.size / totalModules
	renderedSize := totalModules * scale
	outerPadding := (cfg.size - renderedSize) / 2
	return renderLayout{
		scale:       scale,
		moduleStart: outerPadding + cfg.margin*scale,
	}, nil
}

func renderImage(ctx context.Context, matrix [][]bool, cfg options) (*image.NRGBA, error) {
	layout, err := calculateLayout(matrix, cfg)
	if err != nil {
		return nil, err
	}
	background := cfg.background
	if cfg.transparent {
		background.A = 0
	}
	canvas := image.NewNRGBA(image.Rect(0, 0, cfg.size, cfg.size))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)

	foreground := image.NewUniform(cfg.foreground)
	for y, row := range matrix {
		if y%16 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for x, dark := range row {
			if !dark {
				continue
			}
			left := layout.moduleStart + x*layout.scale
			top := layout.moduleStart + y*layout.scale
			rect := image.Rect(left, top, left+layout.scale, top+layout.scale)
			draw.Draw(canvas, rect, foreground, image.Point{}, draw.Src)
		}
	}

	logo, err := prepareLogo(cfg)
	if err != nil {
		return nil, err
	}
	if logo != nil {
		point := image.Pt((cfg.size-logo.Bounds().Dx())/2, (cfg.size-logo.Bounds().Dy())/2)
		draw.Draw(canvas, logo.Bounds().Add(point), logo, image.Point{}, draw.Over)
	}
	return canvas, nil
}

func encodePNG(image image.Image, maxBytes int) ([]byte, error) {
	var buffer bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := encoder.Encode(&buffer, image); err != nil {
		return nil, fmt.Errorf("qrcode: encode png: %w", err)
	}
	if buffer.Len() > maxBytes {
		return nil, fmt.Errorf("%w: bytes=%d limit=%d", ErrOutputTooLarge, buffer.Len(), maxBytes)
	}
	return buffer.Bytes(), nil
}

func renderSVG(ctx context.Context, matrix [][]bool, cfg options) ([]byte, error) {
	layout, err := calculateLayout(matrix, cfg)
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	buffer.Grow(len(matrix) * len(matrix) * 8)
	buffer.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="`)
	buffer.WriteString(strconv.Itoa(cfg.size))
	buffer.WriteString(`" height="`)
	buffer.WriteString(strconv.Itoa(cfg.size))
	buffer.WriteString(`" viewBox="0 0 `)
	buffer.WriteString(strconv.Itoa(cfg.size))
	buffer.WriteByte(' ')
	buffer.WriteString(strconv.Itoa(cfg.size))
	buffer.WriteString(`" shape-rendering="crispEdges">`)

	background := cfg.background
	if cfg.transparent {
		background.A = 0
	}
	if background.A > 0 {
		buffer.WriteString(`<rect width="100%" height="100%"`)
		writeSVGFill(&buffer, background)
		buffer.WriteString(`/>`)
	}
	buffer.WriteString(`<path`)
	writeSVGFill(&buffer, cfg.foreground)
	buffer.WriteString(` d="`)
	for y, row := range matrix {
		if y%16 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		for x, dark := range row {
			if !dark {
				continue
			}
			left := layout.moduleStart + x*layout.scale
			top := layout.moduleStart + y*layout.scale
			fmt.Fprintf(&buffer, "M%d %dh%dv%dh-%dz", left, top, layout.scale, layout.scale, layout.scale)
		}
	}
	buffer.WriteString(`"/>`)

	logo, err := prepareLogo(cfg)
	if err != nil {
		return nil, err
	}
	if logo != nil {
		logoPNG, err := encodePNG(logo, cfg.maxOutputBytes)
		if err != nil {
			return nil, err
		}
		left := (cfg.size - logo.Bounds().Dx()) / 2
		top := (cfg.size - logo.Bounds().Dy()) / 2
		fmt.Fprintf(&buffer, `<image x="%d" y="%d" width="%d" height="%d" href="data:image/png;base64,%s"/>`,
			left, top, logo.Bounds().Dx(), logo.Bounds().Dy(), base64.StdEncoding.EncodeToString(logoPNG))
	}
	buffer.WriteString(`</svg>`)
	if buffer.Len() > cfg.maxOutputBytes {
		return nil, fmt.Errorf("%w: bytes=%d limit=%d", ErrOutputTooLarge, buffer.Len(), cfg.maxOutputBytes)
	}
	return buffer.Bytes(), nil
}

func writeSVGFill(buffer *bytes.Buffer, value color.NRGBA) {
	fmt.Fprintf(buffer, ` fill="#%02x%02x%02x"`, value.R, value.G, value.B)
	if value.A < 0xff {
		fmt.Fprintf(buffer, ` fill-opacity="%.4f"`, float64(value.A)/255)
	}
}
