package service

import (
	"0chain.net/internal/infrastrcture/service/logger"
	"context"
)

type Option func(*Options)

type Options struct {
	Name    string
	Version string
	Context context.Context

	Logger logger.Logger
}

func newOptions(opts ...Option) Options {
	opt := Options{
		Context: context.Background(),
	}

	for _, o := range opts {
		o(&opt)
	}

	return opt
}

func Name(name string) Option {
	return func(o *Options) {
		o.Name = name
	}
}

func Version(version string) Option {
	return func(o *Options) {
		o.Version = version
	}
}

func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}

func Logger(l logger.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}
