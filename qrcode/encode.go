package qrcode

import (
	"context"
	"fmt"
	"strings"

	skipqrcode "github.com/skip2/go-qrcode"
)

func encodeMatrix(ctx context.Context, content string, cfg options) (matrix [][]bool, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			matrix = nil
			err = fmt.Errorf("%w: backend panic: %v", ErrEncodingFailed, recovered)
		}
	}()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if content == "" {
		return nil, ErrEmptyContent
	}
	if len(content) > cfg.maxContentBytes {
		return nil, fmt.Errorf("%w: bytes=%d safety_limit=%d", ErrContentTooLong, len(content), cfg.maxContentBytes)
	}

	code, err := skipqrcode.New(content, cfg.level.backend())
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "too long") || strings.Contains(strings.ToLower(err.Error()), "too large") {
			return nil, fmt.Errorf("%w: %v", ErrContentTooLong, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrEncodingFailed, err)
	}
	code.DisableBorder = true
	matrix = code.Bitmap()
	if len(matrix) == 0 {
		return nil, fmt.Errorf("%w: backend returned an empty matrix", ErrEncodingFailed)
	}
	for _, row := range matrix {
		if len(row) != len(matrix) {
			return nil, fmt.Errorf("%w: backend returned a non-square matrix", ErrEncodingFailed)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return matrix, nil
}
