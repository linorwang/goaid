package qrcode

import "fmt"

// Format identifies an encoded QR Code output format.
type Format uint8

const (
	FormatPNG Format = iota + 1
	FormatSVG
)

func (f Format) String() string {
	switch f {
	case FormatPNG:
		return "png"
	case FormatSVG:
		return "svg"
	default:
		return "unknown"
	}
}

// MIMEType returns the media type for the format.
func (f Format) MIMEType() string {
	switch f {
	case FormatPNG:
		return "image/png"
	case FormatSVG:
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}

func (f Format) validate() error {
	if f != FormatPNG && f != FormatSVG {
		return fmt.Errorf("%w: %d", ErrUnsupportedFormat, f)
	}
	return nil
}
