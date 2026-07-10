package qrcode

import (
	"fmt"

	skipqrcode "github.com/skip2/go-qrcode"
)

// ErrorCorrectionLevel controls how much damage a QR Code may recover from.
type ErrorCorrectionLevel uint8

const (
	ErrorCorrectionLow ErrorCorrectionLevel = iota + 1
	ErrorCorrectionMedium
	ErrorCorrectionQuartile
	ErrorCorrectionHigh
)

func (l ErrorCorrectionLevel) validate() error {
	if l < ErrorCorrectionLow || l > ErrorCorrectionHigh {
		return fmt.Errorf("%w: %d", ErrInvalidCorrection, l)
	}
	return nil
}

func (l ErrorCorrectionLevel) backend() skipqrcode.RecoveryLevel {
	switch l {
	case ErrorCorrectionLow:
		return skipqrcode.Low
	case ErrorCorrectionQuartile:
		return skipqrcode.High
	case ErrorCorrectionHigh:
		return skipqrcode.Highest
	default:
		return skipqrcode.Medium
	}
}
