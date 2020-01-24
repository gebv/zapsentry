package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap/zapcore"
)

// Configuration is a minimal set of parameters for Sentry integration.
type Configuration struct {
	ClientOptions sentry.ClientOptions
	Tags          map[string]string

	TraceSkipFrames int
	Stacktrace      bool
	LevelEnabler    zapcore.Level
	FlushTimeout    time.Duration
	Hub             *sentry.Hub
}

type Option func(*Configuration)

func TraceSkipFrames(i int) Option {
	return func(opt *Configuration) {
		opt.TraceSkipFrames = i
	}
}

func WithSentry(dsn string, tags map[string]string) Option {
	return func(opt *Configuration) {
		opt.ClientOptions.Dsn = dsn
		SentryTags(tags)(opt)
	}
}

// Enviroment sets enviroment.
func Enviroment(env string) Option {
	return func(opt *Configuration) {
		opt.ClientOptions.Environment = env
	}
}

// Release sets release.
func Release(release string) Option {
	return func(opt *Configuration) {
		opt.ClientOptions.Release = release
	}
}

func IgnoreErrors(ignoreErrors []string) Option {
	return func(opt *Configuration) {
		opt.ClientOptions.IgnoreErrors = ignoreErrors
	}
}

// SentryTags sets tags.
func SentryTags(tags map[string]string) Option {
	return func(opt *Configuration) {
		if opt.Tags == nil {
			opt.Tags = make(map[string]string)
		}
		for k, v := range tags {
			opt.Tags[k] = v
		}
	}
}

// SentryTag add one tag.
func SentryTag(key, val string) Option {
	return func(opt *Configuration) {
		if opt.Tags == nil {
			opt.Tags = make(map[string]string)
		}
		opt.Tags[key] = val
	}
}

func DisableStacktrace() Option {
	return func(opt *Configuration) {
		opt.Stacktrace = false
	}
}

func EnableStacktrace() Option {
	return func(opt *Configuration) {
		opt.Stacktrace = true
	}
}

func LevelEnabler(level zapcore.Level) Option {
	return func(opt *Configuration) {
		opt.LevelEnabler = level
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
