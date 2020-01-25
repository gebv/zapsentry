package zapsentry

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// DefaultOptions commonly used options.
var DefaultOptions = []Option{
	LevelEnabler(zap.ErrorLevel),
	TraceSkipFrames(3), // skip 3 first layers in stacktraces
	EnableStacktrace(),
}

// SetWith returns logger with sentry client.
func SetWith(l *zap.Logger, cfg Configuration, c *sentry.Client) (*zap.Logger, error) {
	if c != nil {
		return l.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, NewCore(cfg, c))
		})), nil
	}

	return build(l, cfg)
}

// Set returns logger with sentry client.
func Set(l *zap.Logger, opts ...Option) (*zap.Logger, error) {
	cfg := &Configuration{}
	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.ClientOptions.Dsn == "" || cfg.ClientOptions.Dsn == "test" {
		return l, nil
	}

	return build(l, *cfg)
}

func build(l *zap.Logger, cfg Configuration) (*zap.Logger, error) {
	var sentryCore zapcore.Core
	client, err := sentry.NewClient(cfg.ClientOptions)
	if err != nil {
		sentryCore = zapcore.NewNopCore()
	} else {
		sentryCore = NewCore(cfg, client)
	}

	return l.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, sentryCore)
	})), err
}

// NewCore returns implementation of zapcore.Core.
func NewCore(cfg Configuration, client *sentry.Client) *core {
	core := &core{
		client:       client,
		cfg:          &cfg,
		LevelEnabler: cfg.LevelEnabler,
		flushTimeout: 5 * time.Second,
		fields:       make(map[string]interface{}),
	}

	if cfg.FlushTimeout > 0 {
		core.flushTimeout = cfg.FlushTimeout
	}

	return core
}

func (c *core) With(fs []zapcore.Field) zapcore.Core {
	return c.with(fs)
}

func (c *core) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.cfg.LevelEnabler.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *core) Write(ent zapcore.Entry, fs []zapcore.Field) error {
	clone := c.with(fs)

	event := sentry.NewEvent()
	event.Message = ent.Message
	event.Timestamp = ent.Time.Unix()
	event.Level = sentrySeverity(ent.Level)
	event.Platform = "go"
	event.Extra = clone.fields
	event.Tags = c.cfg.Tags

	if c.cfg.Stacktrace {
		trace := sentry.NewStacktrace()

		if trace != nil {
			if c.cfg.TraceSkipFrames > 0 && len(trace.Frames) >= c.cfg.TraceSkipFrames {
				trace.Frames = trace.Frames[:len(trace.Frames)-c.cfg.TraceSkipFrames]
			}

			event.Exception = []sentry.Exception{{
				Type:       "ValueError",
				Value:      ent.Message,
				Module:     ent.Caller.TrimmedPath(),
				Stacktrace: trace,
			}}
		}
	}

	hub := c.cfg.Hub
	if hub == nil {
		hub = sentry.CurrentHub()
	}
	_ = c.client.CaptureEvent(event, nil, hub.Scope())

	// We may be crashing the program, so should flush any buffered events.
	if ent.Level > zapcore.ErrorLevel {
		c.client.Flush(c.flushTimeout)
	}
	return nil
}

func (c *core) Sync() error {
	c.client.Flush(c.flushTimeout)
	return nil
}

func (c *core) with(fs []zapcore.Field) *core {
	// Copy our map.
	m := make(map[string]interface{}, len(c.fields))
	for k, v := range c.fields {
		m[k] = v
	}

	// Add fields to an in-memory encoder.
	enc := zapcore.NewMapObjectEncoder()
	for _, f := range fs {
		f.AddTo(enc)
	}

	// Merge the two maps.
	for k, v := range enc.Fields {
		m[k] = v
	}

	return &core{
		client:       c.client,
		cfg:          c.cfg,
		fields:       m,
		LevelEnabler: c.LevelEnabler,
	}
}

type core struct {
	client *sentry.Client
	cfg    *Configuration
	zapcore.LevelEnabler
	flushTimeout time.Duration

	fields map[string]interface{}
}

// check interface
var _ zapcore.Core = (*core)(nil)

func sentrySeverity(lvl zapcore.Level) sentry.Level {
	switch lvl {
	case zapcore.DebugLevel:
		return sentry.LevelDebug
	case zapcore.InfoLevel:
		return sentry.LevelInfo
	case zapcore.WarnLevel:
		return sentry.LevelWarning
	case zapcore.ErrorLevel:
		return sentry.LevelError
	case zapcore.DPanicLevel:
		return sentry.LevelFatal
	case zapcore.PanicLevel:
		return sentry.LevelFatal
	case zapcore.FatalLevel:
		return sentry.LevelFatal
	default:
		// Unrecognized levels are fatal.
		return sentry.LevelFatal
	}
}
