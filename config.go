package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	Tags              map[string]string
	DisableStacktrace bool
	Enviroment        string
	Release           string
	Level             zapcore.Level
	FlushTimeout      time.Duration
	Hub               *sentry.Hub
}

type Option func(*Configuration)

func Enviroment(env string) Option {
	return func(opt *Configuration) {
		opt.Enviroment = env
	}
}

func Release(release string) Option {
	return func(opt *Configuration) {
		opt.Release = release
	}
}

func AddTag(key, val string) Option {
	return func(opt *Configuration) {
		if opt.Tags == nil {
			opt.Tags = make(map[string]string)
		}
		opt.Tags[key] = val
	}
}

func DisableStacktrace() Option {
	return func(opt *Configuration) {
		opt.DisableStacktrace = true
	}
}

func EnableStacktrace() Option {
	return func(opt *Configuration) {
		opt.DisableStacktrace = false
	}
}

func ErrorLevel(level zapcore.Level) Option {
	return func(opt *Configuration) {
		opt.Level = level
	}
}

func FlushTimeout(d time.Duration) Option {
	return func(opt *Configuration) {
		opt.FlushTimeout = d
	}
}

func SentryHub(hub *sentry.Hub) Option {
	return func(opt *Configuration) {
		opt.Hub = hub
	}
}
