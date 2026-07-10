package qrcode

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io/fs"
	"reflect"
)

const (
	DefaultSize             = 256
	DefaultMargin           = 4
	DefaultMaxContentBytes  = 16 * 1024
	DefaultMaxImageSize     = 4096
	DefaultMaxLogoBytes     = int64(5 * 1024 * 1024)
	DefaultMaxLogoDimension = 4096
	DefaultMaxOutputBytes   = 32 * 1024 * 1024
	DefaultLogoRatio        = 0.18
	MaxLogoRatio            = 0.25

	minimumImageSize = 64
)

// Option configures QR Code generation.
type Option func(*options) error

type logoSourceKind uint8

const (
	logoSourceNone logoSourceKind = iota
	logoSourceImage
	logoSourceBytes
	logoSourceFile
)

type options struct {
	size             int
	margin           int
	format           Format
	level            ErrorCorrectionLevel
	foreground       color.NRGBA
	background       color.NRGBA
	transparent      bool
	maxContentBytes  int
	maxImageSize     int
	maxLogoBytes     int64
	maxLogoDimension int
	maxOutputBytes   int

	logoKind         logoSourceKind
	logoImage        *image.NRGBA
	logoBytes        []byte
	logoPath         string
	logoRatio        float64
	logoPadding      int
	logoCornerRadius int
	logoBackground   color.NRGBA

	fileMode        fs.FileMode
	dirMode         fs.FileMode
	overwrite       bool
	createParentDir bool
}

func defaultOptions() options {
	return options{
		size:             DefaultSize,
		margin:           DefaultMargin,
		format:           FormatPNG,
		level:            ErrorCorrectionMedium,
		foreground:       color.NRGBA{A: 0xff},
		background:       color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		maxContentBytes:  DefaultMaxContentBytes,
		maxImageSize:     DefaultMaxImageSize,
		maxLogoBytes:     DefaultMaxLogoBytes,
		maxLogoDimension: DefaultMaxLogoDimension,
		maxOutputBytes:   DefaultMaxOutputBytes,
		logoRatio:        DefaultLogoRatio,
		logoPadding:      4,
		logoCornerRadius: 8,
		logoBackground:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		fileMode:         0o644,
		dirMode:          0o755,
	}
}

func WithSize(size int) Option {
	return func(cfg *options) error {
		cfg.size = size
		return nil
	}
}

// WithMargin sets the quiet zone in QR modules. Four modules is recommended.
func WithMargin(modules int) Option {
	return func(cfg *options) error {
		cfg.margin = modules
		return nil
	}
}

func WithFormat(format Format) Option {
	return func(cfg *options) error {
		cfg.format = format
		return nil
	}
}

func WithErrorCorrection(level ErrorCorrectionLevel) Option {
	return func(cfg *options) error {
		cfg.level = level
		return nil
	}
}

func WithForeground(value color.Color) Option {
	return func(cfg *options) error {
		converted, err := normalizedColor(value)
		if err != nil {
			return fmt.Errorf("%w: foreground", err)
		}
		cfg.foreground = converted
		return nil
	}
}

func WithBackground(value color.Color) Option {
	return func(cfg *options) error {
		converted, err := normalizedColor(value)
		if err != nil {
			return fmt.Errorf("%w: background", err)
		}
		cfg.background = converted
		return nil
	}
}

func WithTransparent(transparent bool) Option {
	return func(cfg *options) error {
		cfg.transparent = transparent
		return nil
	}
}

func WithLogoImage(value image.Image) Option {
	logo, copyErr := cloneImageWithinLimit(value, DefaultMaxLogoDimension)
	return func(cfg *options) error {
		if copyErr != nil {
			return copyErr
		}
		cfg.logoKind = logoSourceImage
		cfg.logoImage = logo
		cfg.logoBytes = nil
		cfg.logoPath = ""
		return nil
	}
}

func WithLogoBytes(value []byte) Option {
	copied := append([]byte(nil), value...)
	return func(cfg *options) error {
		if len(copied) == 0 {
			return fmt.Errorf("%w: bytes are empty", ErrInvalidLogo)
		}
		cfg.logoKind = logoSourceBytes
		cfg.logoImage = nil
		cfg.logoBytes = copied
		cfg.logoPath = ""
		return nil
	}
}

// WithLogoFile loads a local PNG or JPEG logo. It never performs network IO.
func WithLogoFile(filename string) Option {
	return func(cfg *options) error {
		if filename == "" {
			return fmt.Errorf("%w: filename is empty", ErrInvalidLogo)
		}
		cfg.logoKind = logoSourceFile
		cfg.logoImage = nil
		cfg.logoBytes = nil
		cfg.logoPath = filename
		return nil
	}
}

func WithLogoRatio(ratio float64) Option {
	return func(cfg *options) error {
		cfg.logoRatio = ratio
		return nil
	}
}

// WithLogoPadding sets the background plate padding in output pixels.
func WithLogoPadding(pixels int) Option {
	return func(cfg *options) error {
		cfg.logoPadding = pixels
		return nil
	}
}

// WithLogoCornerRadius sets the logo corner radius in output pixels.
func WithLogoCornerRadius(pixels int) Option {
	return func(cfg *options) error {
		cfg.logoCornerRadius = pixels
		return nil
	}
}

func WithLogoBackground(value color.Color) Option {
	return func(cfg *options) error {
		converted, err := normalizedColor(value)
		if err != nil {
			return fmt.Errorf("%w: logo background", err)
		}
		cfg.logoBackground = converted
		return nil
	}
}

func WithMaxContentBytes(limit int) Option {
	return func(cfg *options) error {
		cfg.maxContentBytes = limit
		return nil
	}
}

func WithMaxImageSize(limit int) Option {
	return func(cfg *options) error {
		cfg.maxImageSize = limit
		return nil
	}
}

func WithMaxLogoBytes(limit int64) Option {
	return func(cfg *options) error {
		cfg.maxLogoBytes = limit
		return nil
	}
}

func WithMaxLogoDimension(limit int) Option {
	return func(cfg *options) error {
		cfg.maxLogoDimension = limit
		return nil
	}
}

func WithMaxOutputBytes(limit int) Option {
	return func(cfg *options) error {
		cfg.maxOutputBytes = limit
		return nil
	}
}

func WithFileMode(mode fs.FileMode) Option {
	return func(cfg *options) error {
		cfg.fileMode = mode
		return nil
	}
}

func WithDirMode(mode fs.FileMode) Option {
	return func(cfg *options) error {
		cfg.dirMode = mode
		return nil
	}
}

func WithOverwrite(overwrite bool) Option {
	return func(cfg *options) error {
		cfg.overwrite = overwrite
		return nil
	}
}

func WithCreateParentDir(create bool) Option {
	return func(cfg *options) error {
		cfg.createParentDir = create
		return nil
	}
}

func applyOptions(opts []Option) (options, error) {
	cfg := defaultOptions()
	for index, opt := range opts {
		if opt == nil {
			return options{}, fmt.Errorf("%w: option %d is nil", ErrInvalidOption, index)
		}
		if err := opt(&cfg); err != nil {
			return options{}, fmt.Errorf("qrcode: apply option %d: %w", index, err)
		}
	}
	if err := cfg.finalize(); err != nil {
		return options{}, err
	}
	return cfg, nil
}

func (cfg *options) finalize() error {
	if cfg.size < minimumImageSize || cfg.maxImageSize < minimumImageSize || cfg.size > cfg.maxImageSize {
		return fmt.Errorf("%w: size=%d allowed=%d..%d", ErrInvalidSize, cfg.size, minimumImageSize, cfg.maxImageSize)
	}
	if cfg.margin < 0 || cfg.margin > 64 {
		return fmt.Errorf("%w: %d", ErrInvalidMargin, cfg.margin)
	}
	if err := cfg.format.validate(); err != nil {
		return err
	}
	if err := cfg.level.validate(); err != nil {
		return err
	}
	if cfg.foreground.A == 0 {
		return fmt.Errorf("%w: foreground is fully transparent", ErrInvalidColor)
	}
	background := cfg.background
	if cfg.transparent {
		background.A = 0
	}
	if !cfg.transparent && cfg.foreground == background {
		return fmt.Errorf("%w: foreground and background are identical", ErrInvalidColor)
	}
	if cfg.maxContentBytes <= 0 || cfg.maxImageSize <= 0 || cfg.maxLogoBytes <= 0 || cfg.maxLogoDimension <= 0 || cfg.maxOutputBytes <= 0 {
		return fmt.Errorf("%w: resource limits must be positive", ErrInvalidOption)
	}
	if cfg.fileMode.Perm() == 0 || cfg.fileMode&^fs.ModePerm != 0 {
		return fmt.Errorf("%w: invalid file mode %v", ErrInvalidOption, cfg.fileMode)
	}
	if cfg.dirMode.Perm() == 0 || cfg.dirMode&^fs.ModePerm != 0 {
		return fmt.Errorf("%w: invalid directory mode %v", ErrInvalidOption, cfg.dirMode)
	}
	if cfg.logoKind == logoSourceNone {
		return nil
	}
	if cfg.logoRatio <= 0 || cfg.logoRatio > MaxLogoRatio {
		return fmt.Errorf("%w: ratio %.4f allowed=(0, %.2f]", ErrLogoTooLarge, cfg.logoRatio, MaxLogoRatio)
	}
	if cfg.logoPadding < 0 || cfg.logoCornerRadius < 0 {
		return fmt.Errorf("%w: negative padding or corner radius", ErrInvalidLogo)
	}
	plateRatio := cfg.logoRatio + 2*float64(cfg.logoPadding)/float64(cfg.size)
	if plateRatio > MaxLogoRatio {
		return fmt.Errorf("%w: logo and padding ratio %.4f exceeds %.2f", ErrLogoTooLarge, plateRatio, MaxLogoRatio)
	}
	logo, err := cfg.loadLogo()
	if err != nil {
		return err
	}
	cfg.logoImage = logo
	cfg.logoBytes = nil
	cfg.logoPath = ""
	cfg.logoKind = logoSourceImage
	cfg.level = ErrorCorrectionHigh
	return nil
}

func normalizedColor(value color.Color) (color.NRGBA, error) {
	if isNil(value) {
		return color.NRGBA{}, ErrInvalidColor
	}
	converted, ok := color.NRGBAModel.Convert(value).(color.NRGBA)
	if !ok {
		return color.NRGBA{}, ErrInvalidColor
	}
	return converted, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

func cloneImage(source image.Image) (*image.NRGBA, error) {
	if isNil(source) {
		return nil, fmt.Errorf("%w: image is nil", ErrInvalidLogo)
	}
	bounds := source.Bounds()
	if bounds.Empty() {
		return nil, fmt.Errorf("%w: image bounds are empty", ErrInvalidLogo)
	}
	destination := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(destination, destination.Bounds(), source, bounds.Min, draw.Src)
	return destination, nil
}
