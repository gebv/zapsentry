package zapsentry

// tests copy and adapted from https://github.com/plimble/zap-sentry/blob/master/sentry_test.go

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestSentrySeverityMap(t *testing.T) {
	tests := []struct {
		z zapcore.Level
		r sentry.Level
	}{
		{zap.DebugLevel, sentry.LevelDebug},
		{zap.InfoLevel, sentry.LevelInfo},
		{zap.WarnLevel, sentry.LevelWarning},
		{zap.ErrorLevel, sentry.LevelError},
		{zap.DPanicLevel, sentry.LevelFatal},
		{zap.PanicLevel, sentry.LevelFatal},
		{zap.FatalLevel, sentry.LevelFatal},
		{zapcore.Level(-42), sentry.LevelFatal},
		{zapcore.Level(100), sentry.LevelFatal},
	}

	for _, tt := range tests {
		assert.Equal(
			t,
			tt.r,
			sentrySeverity(tt.z),
			"Unexpected output converting zap Level %s to raven Severity.", tt.z,
		)
	}
}

func TestCoreWith(t *testing.T) {

	// Ensure that we're not sharing map references across generations.
	parent := newCore(Configuration{}, nil).With([]zapcore.Field{zap.String("parent", "parent")})
	elder := parent.With([]zapcore.Field{zap.String("elder", "elder")})
	younger := parent.With([]zapcore.Field{zap.String("younger", "younger")})

	parentC := assertCore(t, parent)
	elderC := assertCore(t, elder)
	youngerC := assertCore(t, younger)

	assert.Equal(t, map[string]interface{}{
		"parent": "parent",
	}, parentC.fields, "Unexpected fields on parent.")
	assert.Equal(t, map[string]interface{}{
		"parent": "parent",
		"elder":  "elder",
	}, elderC.fields, "Unexpected fields on first child core.")
	assert.Equal(t, map[string]interface{}{
		"parent":  "parent",
		"younger": "younger",
	}, youngerC.fields, "Unexpected fields on second child core.")
}

func TestCoreCheck(t *testing.T) {
	core := newCore(Configuration{
		LevelEnabler: zapcore.ErrorLevel,
	}, nil)
	assert.Nil(t, core.Check(zapcore.Entry{}, nil), "Expected nil CheckedEntry for disabled levels.")
	ent := zapcore.Entry{Level: zapcore.ErrorLevel}
	assert.NotNil(t, core.Check(ent, nil), "Expected non-nil CheckedEntry for enabled levels.")
}

func TestConfigWrite(t *testing.T) {
	client, transport := setupClientTest()
	require.NotNil(t, client)
	core := newCore(Configuration{
		LevelEnabler:    zapcore.ErrorLevel,
		TraceSkipFrames: 2,
	}, client)
	require.NotNil(t, core.client)

	// Write a panic-level message, which should also fire a Sentry event.
	ent := zapcore.Entry{Message: "oh no", Level: zapcore.PanicLevel, Time: time.Now()}
	ce := core.With([]zapcore.Field{zap.String("foo", "bar")}).Check(ent, nil)
	require.NotNil(t, ce, "Expected Check to return non-nil CheckedEntry at enabled levels.")
	ce.Write(zap.String("bar", "baz"))

	// Assert that we wrote and flushed a packet.
	require.Equal(t, 1, len(transport.events), "Expected to write one Sentry packet.")

	// Assert that the captured packet is shaped correctly.
	p := transport.events[0]
	assert.Equal(t, "oh no", p.Message, "Unexpected message in captured packet.")
	assert.Equal(t, sentry.LevelFatal, p.Level, "Unexpected severity in captured packet.")
	require.Equal(t, 1, len(p.Exception), "Expected a stacktrace in packet interfaces.")
	trace := p.Exception[0].Stacktrace
	require.NotNil(t, trace, "Expected only interface in packet to be a stacktrace.")
	// Trace should contain this test and testing harness main.
	require.Equal(t, 1, len(trace.Frames), "Expected stacktrace to contain at least two frame.")

	frame := trace.Frames[0]
	assert.Equal(t, "TestConfigWrite", frame.Function, "Expected frame to point to this test function.")
}

func TestConfigBuild(t *testing.T) {
	_, err := Set(zap.NewNop(), WithSentry("invalid", map[string]string{}))
	assert.Error(t, err, "Expected invalid DSN to make config building fail.")
}

func TestStackTraces(t *testing.T) {
	client, transport := setupClientTest()
	require.NotNil(t, client)
	core := newCore(Configuration{
		LevelEnabler:    zapcore.ErrorLevel,
		TraceSkipFrames: 2,
	}, client)
	require.NotNil(t, core.client)

	err1 := io.EOF
	// err2 := errors.Wrap(err1, "second error")
	// err3 := errors.WithStack(err2)
	// err4 := errors.WithMessage(err3, "fourth error")
	err5 := errors.Wrap(err1, "fifth error")

	l := zap.New(core)
	l.Error("Log message", zap.Error(err5))

	require.Len(t, transport.events, 1)
	actual := transport.events[0]
	expected := &sentry.Event{
		Message:   "Log message",
		Timestamp: actual.Timestamp,
		Level:     "error",
		Platform:  "go",
		Extra: map[string]interface{}{
			"error":        err5.Error(),
			"errorVerbose": fmt.Sprintf("%+v", err5),
		},
		Exception: []sentry.Exception{
			{
				Type:   "ValueError",
				Value:  "Log message",
				Module: "undefined",
				Stacktrace: &sentry.Stacktrace{
					Frames: []sentry.Frame{
						{
							Filename: "sentry_test.go",
							Function: "TestStackTraces",
							Module:   "github.com/TheZeroSlave/zapsentry",
						},
						{
							Filename: "logger.go",
							Function: "go.uber.org/zap.(*Logger).Error",
							Module:   "",
						},
					},
				},
			},
		},
	}

	assert.Equal(t, expected.Message, actual.Message)
	assert.Equal(t, expected.Timestamp, actual.Timestamp)
	assert.Equal(t, expected.Level, actual.Level)
	assert.Equal(t, expected.Platform, actual.Platform)
	assert.Equal(t, expected.Extra, actual.Extra)

	require.Len(t, expected.Exception, 1)
	assert.Equal(t, expected.Exception[0].Type, actual.Exception[0].Type)
	assert.Equal(t, expected.Exception[0].Value, actual.Exception[0].Value)
	assert.Equal(t, expected.Exception[0].Module, actual.Exception[0].Module)

	require.NotNil(t, expected.Exception[0].Stacktrace)
	require.Len(t, expected.Exception[0].Stacktrace.Frames, 2)
	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[0].Filename, actual.Exception[0].Stacktrace.Frames[0].Filename)
	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[0].Function, actual.Exception[0].Stacktrace.Frames[0].Function)
	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[0].Module, actual.Exception[0].Stacktrace.Frames[0].Module)

	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[1].Filename, actual.Exception[0].Stacktrace.Frames[1].Filename)
	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[1].Function, actual.Exception[0].Stacktrace.Frames[1].Function)
	assert.Equal(t, expected.Exception[0].Stacktrace.Frames[1].Module, actual.Exception[0].Stacktrace.Frames[1].Module)
}

func assertCore(t testing.TB, iface zapcore.Core) *core {
	c, ok := iface.(*core)
	require.True(t, ok, "Failed to cast Core to sentry *core.")
	return c
}

// copy from github.com/getsentry/sentry-go@v0.4.0/mocks_test.go
func setupClientTest() (*sentry.Client, *TransportMock) {
	transport := &TransportMock{}
	client, _ := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "http://whatever@really.com/1337",
		Transport: transport,
		Integrations: func(i []sentry.Integration) []sentry.Integration {
			return []sentry.Integration{}
		},
	})

	return client, transport
}

type TransportMock struct {
	mu        sync.Mutex
	events    []*sentry.Event
	lastEvent *sentry.Event
}

func (t *TransportMock) Configure(options sentry.ClientOptions) {}
func (t *TransportMock) SendEvent(event *sentry.Event) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
	t.lastEvent = event
}
func (t *TransportMock) Flush(timeout time.Duration) bool {
	return true
}
func (t *TransportMock) Events() []*sentry.Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.events
}
