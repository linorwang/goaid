package qrcode

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"
)

func (cfg *options) loadLogo() (*image.NRGBA, error) {
	var logo *image.NRGBA
	var err error

	switch cfg.logoKind {
	case logoSourceImage:
		logo = cfg.logoImage
	case logoSourceBytes:
		logo, err = cfg.decodeLogo(cfg.logoBytes)
	case logoSourceFile:
		logo, err = cfg.loadLogoFile(cfg.logoPath)
	default:
		return nil, fmt.Errorf("%w: unsupported source", ErrInvalidLogo)
	}
	if err != nil {
		return nil, err
	}
	if logo == nil || logo.Bounds().Empty() {
		return nil, fmt.Errorf("%w: image is empty", ErrInvalidLogo)
	}
	if logo.Bounds().Dx() > cfg.maxLogoDimension || logo.Bounds().Dy() > cfg.maxLogoDimension {
		return nil, fmt.Errorf("%w: dimensions=%dx%d limit=%d", ErrLogoTooLarge, logo.Bounds().Dx(), logo.Bounds().Dy(), cfg.maxLogoDimension)
	}
	return logo, nil
}

func (cfg *options) loadLogoFile(filename string) (*image.NRGBA, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("%w: open %q: %v", ErrInvalidLogo, filename, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: stat %q: %v", ErrInvalidLogo, filename, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %q is not a regular file", ErrInvalidLogo, filename)
	}
	if info.Size() > cfg.maxLogoBytes {
		return nil, fmt.Errorf("%w: bytes=%d limit=%d", ErrLogoTooLarge, info.Size(), cfg.maxLogoBytes)
	}

	reader := io.LimitReader(file, cfg.maxLogoBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%w: read %q: %v", ErrInvalidLogo, filename, err)
	}
	return cfg.decodeLogo(data)
}

func (cfg *options) decodeLogo(data []byte) (*image.NRGBA, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("%w: image bytes are empty", ErrInvalidLogo)
	}
	if int64(len(data)) > cfg.maxLogoBytes {
		return nil, fmt.Errorf("%w: bytes=%d limit=%d", ErrLogoTooLarge, len(data), cfg.maxLogoBytes)
	}

	decodedConfig, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: decode config: %v", ErrInvalidLogo, err)
	}
	if decodedConfig.Width <= 0 || decodedConfig.Height <= 0 || decodedConfig.Width > cfg.maxLogoDimension || decodedConfig.Height > cfg.maxLogoDimension {
		return nil, fmt.Errorf("%w: dimensions=%dx%d limit=%d", ErrLogoTooLarge, decodedConfig.Width, decodedConfig.Height, cfg.maxLogoDimension)
	}

	decoded, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: decode image: %v", ErrInvalidLogo, err)
	}
	return cloneImage(decoded)
}

func prepareLogo(cfg options) (*image.NRGBA, error) {
	if cfg.logoImage == nil {
		return nil, nil
	}

	logoSize := int(math.Round(float64(cfg.size) * cfg.logoRatio))
	if logoSize <= 0 {
		return nil, fmt.Errorf("%w: rendered logo size is zero", ErrInvalidLogo)
	}
	plateSize := logoSize + 2*cfg.logoPadding
	if float64(plateSize)/float64(cfg.size) > MaxLogoRatio {
		return nil, fmt.Errorf("%w: rendered plate=%d canvas=%d", ErrLogoTooLarge, plateSize, cfg.size)
	}

	scaled := image.NewNRGBA(image.Rect(0, 0, logoSize, logoSize))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), cfg.logoImage, cfg.logoImage.Bounds(), draw.Src, nil)

	plate := image.NewNRGBA(image.Rect(0, 0, plateSize, plateSize))
	plateMask := roundedMask(plateSize, plateSize, cfg.logoCornerRadius+cfg.logoPadding)
	draw.DrawMask(plate, plate.Bounds(), image.NewUniform(cfg.logoBackground), image.Point{}, plateMask, image.Point{}, draw.Src)

	logoMask := roundedMask(logoSize, logoSize, cfg.logoCornerRadius)
	logoRect := image.Rect(cfg.logoPadding, cfg.logoPadding, cfg.logoPadding+logoSize, cfg.logoPadding+logoSize)
	draw.DrawMask(plate, logoRect, scaled, image.Point{}, logoMask, image.Point{}, draw.Over)
	return plate, nil
}

func roundedMask(width, height, radius int) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	if radius <= 0 {
		draw.Draw(mask, mask.Bounds(), image.NewUniform(color.Alpha{A: 0xff}), image.Point{}, draw.Src)
		return mask
	}
	if radius > width/2 {
		radius = width / 2
	}
	if radius > height/2 {
		radius = height / 2
	}

	radiusSquared := float64(radius * radius)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			inside := x >= radius && x < width-radius || y >= radius && y < height-radius
			if !inside {
				centerX := radius
				if x >= width-radius {
					centerX = width - radius - 1
				}
				centerY := radius
				if y >= height-radius {
					centerY = height - radius - 1
				}
				dx := float64(x - centerX)
				dy := float64(y - centerY)
				inside = dx*dx+dy*dy <= radiusSquared
			}
			if inside {
				mask.SetAlpha(x, y, color.Alpha{A: 0xff})
			}
		}
	}
	return mask
}
