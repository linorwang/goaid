package qrcode

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

func (g *Generator) Save(filename, content string) error {
	return g.SaveContext(context.Background(), filename, content)
}

func (g *Generator) SaveContext(ctx context.Context, filename, content string) error {
	if g == nil {
		return fmt.Errorf("%w: generator is nil", ErrInvalidOption)
	}
	if filename == "" {
		return fmt.Errorf("%w: filename is empty", ErrInvalidPath)
	}
	cleaned := filepath.Clean(filename)
	if cleaned == "." || filepath.Base(cleaned) == "." {
		return fmt.Errorf("%w: %q", ErrInvalidPath, filename)
	}
	data, err := g.GenerateContext(ctx, content)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return atomicWriteFile(ctx, cleaned, data, g.cfg)
}

func atomicWriteFile(ctx context.Context, filename string, data []byte, cfg options) error {
	directory := filepath.Dir(filename)
	if cfg.createParentDir {
		if err := os.MkdirAll(directory, cfg.dirMode); err != nil {
			return fmt.Errorf("qrcode: create output directory %q: %w", directory, err)
		}
	}

	temporary, err := os.CreateTemp(directory, ".qrcode-*.tmp")
	if err != nil {
		return fmt.Errorf("qrcode: create temporary output in %q: %w", directory, err)
	}
	temporaryName := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryName)
		}
	}()

	if err := temporary.Chmod(cfg.fileMode); err != nil {
		return fmt.Errorf("qrcode: set temporary output permissions: %w", err)
	}
	if err := writeAll(temporary, data); err != nil {
		return fmt.Errorf("qrcode: write temporary output: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("qrcode: sync temporary output: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("qrcode: close temporary output: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if cfg.overwrite {
		if err := replaceFile(temporaryName, filename); err != nil {
			return fmt.Errorf("qrcode: replace output %q: %w", filename, err)
		}
	} else {
		if err := os.Link(temporaryName, filename); err != nil {
			if errors.Is(err, fs.ErrExist) {
				return fmt.Errorf("%w: %q", ErrFileExists, filename)
			}
			return fmt.Errorf("qrcode: commit output %q: %w", filename, err)
		}
		if err := os.Remove(temporaryName); err != nil {
			return fmt.Errorf("qrcode: remove temporary output: %w", err)
		}
	}
	committed = true
	if err := syncParentDirectory(directory); err != nil {
		return fmt.Errorf("qrcode: sync output directory %q: %w", directory, err)
	}
	return nil
}

func writeAll(file *os.File, data []byte) error {
	for len(data) > 0 {
		written, err := file.Write(data)
		if err != nil {
			return err
		}
		if written == 0 {
			return fmt.Errorf("zero-byte write: %w", fs.ErrInvalid)
		}
		data = data[written:]
	}
	return nil
}
