package sendsms

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

type clientOptions struct {
	config    *Config
	providers map[string]SMSProvider
	cache     redis.Cmdable
}

// ClientOption configures an SMSClient created by New.
type ClientOption func(*clientOptions) error

// New creates an SMS client with a small, option-based setup API.
func New(options ...ClientOption) (*SMSClient, error) {
	opts := &clientOptions{
		config:    DefaultConfig(),
		providers: make(map[string]SMSProvider),
	}

	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(opts); err != nil {
			return nil, err
		}
	}

	if opts.config.PrimaryProvider == "" && len(opts.providers) == 1 {
		for name := range opts.providers {
			opts.config.PrimaryProvider = name
		}
	}

	return NewSMSClient(
		opts.config.PrimaryProvider,
		opts.config.BackupProviders,
		opts.providers,
		opts.cache,
		opts.config,
	)
}

// WithConfig replaces the default client configuration.
func WithConfig(config *Config) ClientOption {
	return func(opts *clientOptions) error {
		if config == nil {
			opts.config = DefaultConfig()
			return nil
		}
		opts.config = normalizeConfig(config)
		return nil
	}
}

// WithProvider registers a provider by name.
func WithProvider(name string, provider SMSProvider) ClientOption {
	return func(opts *clientOptions) error {
		if name == "" {
			return fmt.Errorf("%w: provider name is empty", ErrConfigInvalid)
		}
		if provider == nil {
			return fmt.Errorf("%w: provider %s is nil", ErrConfigInvalid, name)
		}
		opts.providers[name] = provider
		return nil
	}
}

// WithProviders registers multiple providers.
func WithProviders(providers map[string]SMSProvider) ClientOption {
	return func(opts *clientOptions) error {
		for name, provider := range providers {
			if err := WithProvider(name, provider)(opts); err != nil {
				return err
			}
		}
		return nil
	}
}

// WithPrimary sets the primary provider name.
func WithPrimary(name string) ClientOption {
	return func(opts *clientOptions) error {
		opts.config.PrimaryProvider = name
		return nil
	}
}

// WithBackups sets backup providers used when failover is enabled.
func WithBackups(names ...string) ClientOption {
	return func(opts *clientOptions) error {
		opts.config.BackupProviders = append([]string(nil), names...)
		return nil
	}
}

// WithRedis enables Redis-backed verification code storage and rate limiting.
func WithRedis(cache redis.Cmdable) ClientOption {
	return func(opts *clientOptions) error {
		opts.cache = cache
		return nil
	}
}

// WithFailover toggles failover.
func WithFailover(enabled bool) ClientOption {
	return func(opts *clientOptions) error {
		opts.config.EnableFailover = enabled
		return nil
	}
}

// WithDefaultSign sets the default SMS sign name.
func WithDefaultSign(signName string) ClientOption {
	return func(opts *clientOptions) error {
		opts.config.DefaultSign = signName
		return nil
	}
}

// WithDefaultTemplate sets the default SMS template ID.
func WithDefaultTemplate(templateID string) ClientOption {
	return func(opts *clientOptions) error {
		opts.config.DefaultTemplate = templateID
		return nil
	}
}
