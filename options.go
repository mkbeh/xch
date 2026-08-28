package xch

import (
	"errors"
	"fmt"
	"maps"
	"strings"
)

// Option configures a Pool.
//
// The interface is sealed so options can only be created by this package.
type Option interface {
	apply(*settings) error
}

type optionFunc func(*settings) error

func (option optionFunc) apply(settings *settings) error {
	return option(settings)
}

type settings struct {
	name   string
	labels map[string]string
}

func defaultSettings() *settings {
	return &settings{
		labels: make(map[string]string),
	}
}

func applyOptions(settings *settings, options ...Option) error {
	for _, option := range options {
		if option == nil {
			return errors.New("xch: option is nil")
		}

		if err := option.apply(settings); err != nil {
			return fmt.Errorf("xch: apply option: %w", err)
		}
	}

	return nil
}

// WithName assigns a stable logical name to the pool.
//
// Name is metadata for diagnostics and observability. It does not change the
// underlying ClickHouse connection configuration.
func WithName(name string) Option {
	name = strings.TrimSpace(name)

	return optionFunc(func(settings *settings) error {
		if name == "" {
			return errors.New("pool name must not be blank")
		}

		settings.name = name

		return nil
	})
}

// WithLabel adds or replaces one pool label.
func WithLabel(key, value string) Option {
	return optionFunc(func(settings *settings) error {
		if key == "" {
			return errors.New("label key must not be empty")
		}

		settings.labels[key] = value

		return nil
	})
}

// WithLabels merges labels into the pool metadata.
//
// Labels are defensively copied. When the same key is configured more than
// once, the last value wins.
func WithLabels(labels map[string]string) Option {
	labels = maps.Clone(labels)

	return optionFunc(func(settings *settings) error {
		for key, value := range labels {
			if key == "" {
				return errors.New("label key must not be empty")
			}

			settings.labels[key] = value
		}

		return nil
	})
}
