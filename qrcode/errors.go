package qrcode

import "errors"

var (
	ErrEmptyContent      = errors.New("qrcode: content is empty")
	ErrContentTooLong    = errors.New("qrcode: content is too long")
	ErrInvalidSize       = errors.New("qrcode: invalid size")
	ErrInvalidMargin     = errors.New("qrcode: invalid margin")
	ErrUnsupportedFormat = errors.New("qrcode: unsupported format")
	ErrInvalidCorrection = errors.New("qrcode: invalid error correction level")
	ErrInvalidColor      = errors.New("qrcode: invalid color")
	ErrInvalidOption     = errors.New("qrcode: invalid option")
	ErrEncodingFailed    = errors.New("qrcode: encoding failed")
	ErrInvalidLogo       = errors.New("qrcode: invalid logo")
	ErrLogoTooLarge      = errors.New("qrcode: logo is too large")
	ErrOutputTooLarge    = errors.New("qrcode: output is too large")
	ErrNilWriter         = errors.New("qrcode: writer is nil")
	ErrInvalidPath       = errors.New("qrcode: invalid output path")
	ErrFileExists        = errors.New("qrcode: output file already exists")
	ErrInvalidPayload    = errors.New("qrcode: invalid structured payload")
	ErrImageUnavailable  = errors.New("qrcode: image output is unavailable for this format")
)
