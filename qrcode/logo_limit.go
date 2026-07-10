package qrcode

import (
	"fmt"
	"image"
)

// cloneImageWithinLimit applies an absolute pre-allocation guard. Per-generator
// limits are validated again after options are resolved.
func cloneImageWithinLimit(source image.Image, limit int) (*image.NRGBA, error) {
	if isNil(source) {
		return nil, fmt.Errorf("%w: image is nil", ErrInvalidLogo)
	}
	bounds := source.Bounds()
	if bounds.Empty() {
		return nil, fmt.Errorf("%w: image bounds are empty", ErrInvalidLogo)
	}
	if bounds.Dx() > limit || bounds.Dy() > limit {
		return nil, fmt.Errorf("%w: dimensions=%dx%d hard_limit=%d", ErrLogoTooLarge, bounds.Dx(), bounds.Dy(), limit)
	}
	return cloneImage(source)
}
