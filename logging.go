// Package logging provides structured, process-oriented logging helpers
// built on top of zerolog.
//
// It enforces a consistent event-based log format across the application.
package logging

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

//
// ===== Standard log field names =====
//

const (
	FieldEvent        string = "event"
	FieldResult       string = "result"
	FieldScope        string = "scope"
	FieldSpanID       string = "span_id"
	FieldTraceID      string = "trace_id"
	FieldParams       string = "params"
	FieldDuration     string = "duration"
	FieldParentSpanID string = "parent_span_id"
	FieldAnchor       string = "anchor"
)

// LogContextName is the context.Context key used to store and retrieve
// the current trace identifier.
const LogContextName string = "logging.context"

type LogScope struct {
	Start time.Time
	Log   zerolog.Logger
}

type LogContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
}

func (lc LogContext) Next(spanId string) LogContext {
	return LogContext{
		TraceID:      lc.TraceID,
		SpanID:       spanId,
		ParentSpanID: lc.SpanID,
	}
}

//
// ===== Internal helpers =====
//

// newSpanID generates a short random span identifier (8 hex characters).
func newSpanID() string {
	for {
		b := make([]byte, 8)

		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

		if binary.BigEndian.Uint64(b) != 0 {
			return hex.EncodeToString(b)
		}
	}
}

func newTraceID() string {
	for {
		b := make([]byte, 16)

		_, err := rand.Read(b)
		if err != nil {
			panic(err)
		}

		if binary.BigEndian.Uint64(b[:8]) != 0 ||
			binary.BigEndian.Uint64(b[8:]) != 0 {
			return hex.EncodeToString(b)
		}
	}
}

func RootLogContext() LogContext {
	return LogContext{
		TraceID:      newTraceID(),
		SpanID:       "",
		ParentSpanID: "",
	}
}

// fromCtx resolves the zerolog.Logger stored in the context.
// If no valid logger is found, the global logger is returned.
func fromCtx(ctx context.Context) LogContext {

	if ctx == nil {
		return RootLogContext()
	}
	v := ctx.Value(LogContextName)
	if v == nil {
		return RootLogContext()
	}
	lc, ok := v.(LogContext)
	if !ok {
		return RootLogContext()
	}
	return lc
}

func Detach(ctx context.Context) context.Context {
	out := context.Background()

	logContext := ctx.Value(LogContextName)
	if logContext == nil {
		logContext = RootLogContext()
	}
	return context.WithValue(out, LogContextName, logContext)

}

// AddParams populates the FieldParams log field from the provided parameter map.
//
// Values implementing ObjectWithLevel are marshaled with awareness of
// the current log level.
func AddParams(e *zerolog.Event, level zerolog.Level, params map[string]any) {
	d := zerolog.Dict()
	for k, v := range params {
		if v == nil {
			continue
		}

		if m, ok := v.(ObjectWithLevel); ok {
			d.Object(k, WithLevel(level, m))
			continue
		}

		d.Interface(k, v)
	}
	e.Dict(FieldParams, d)
}

//
// ===== Function-scope logging =====
//

// Enter starts a logical function scope and emits a func.enter event
// at debug or trace log levels.
//
// It returns a span-bound logger for subsequent logging within the function.
func Enter(ctx context.Context, scopeName string, anchor any, params map[string]any) (LogScope, context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	logContext := fromCtx(ctx)

	spanID := newSpanID()

	logg := log.Logger
	w := logg.With().
		Str(FieldTraceID, logContext.TraceID).
		Str(FieldParentSpanID, logContext.SpanID).
		Str(FieldScope, scopeName).
		Str(FieldSpanID, spanID).
		Any(FieldAnchor, anchor)

	l := w.Logger()
	level := globalConfig.Load().resolveLevel(scopeName)
	if level != nil {
		l = l.Level(*level)
	} else {
		le := zerolog.DebugLevel
		level = &le
	}
	e := l.Debug().Str(FieldEvent, "enter")
	if params != nil {
		AddParams(e, *level, params)
	}
	e.Msg("")
	logContext = logContext.Next(spanID)
	ctx = context.WithValue(ctx, LogContextName, logContext)

	return LogScope{
		Log:   l,
		Start: start,
	}, ctx
}

func EnterWithCtx(ctx context.Context, scopeName string, anchor any, params map[string]any) (LogScope, context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now()
	logContext := fromCtx(ctx)

	logg := log.Logger
	w := logg.With().
		Str(FieldTraceID, logContext.TraceID).
		Str(FieldParentSpanID, logContext.ParentSpanID).
		Str(FieldScope, scopeName).
		Str(FieldSpanID, logContext.SpanID).
		Any(FieldAnchor, anchor)
	l := w.Logger()
	level := globalConfig.Load().resolveLevel(scopeName)
	if level != nil {
		l = l.Level(*level)
	}
	e := l.Debug().Str(FieldEvent, "enter")
	if params != nil {
		AddParams(e, *level, params)
	}
	e.Msg("")

	return LogScope{
		Log:   l,
		Start: start,
	}, ctx
}

// Exit closes a function scope with a successful result
// at debug or trace log levels.
func Exit(scope LogScope, result string, params map[string]any) {
	logg := scope.Log
	dur := time.Since(scope.Start)
	e := logg.Debug().
		Str(FieldEvent, "exit").
		Dur(FieldDuration, dur)
	if result != "" {
		e.Str(FieldResult, result)
	}
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	e.Msg("")
}

// ExitWarn closes a function scope with an Warning result.
func ExitWarn(scope LogScope, err error) {
	dur := time.Since(scope.Start)
	scope.Log.Warn().
		Str(FieldEvent, "exit").
		Str(FieldResult, "warning").
		Dur(FieldDuration, dur).
		Err(err).
		Msg("")
}

// ExitErr closes a function scope with an error result.
func ExitErr(scope LogScope, err error) {
	dur := time.Since(scope.Start)
	scope.Log.Error().
		Str(FieldEvent, "exit").
		Str(FieldResult, "error").
		Dur(FieldDuration, dur).
		Err(err).
		Msg("")
}

// ExitErrParams closes a function scope with an error result
// and additional structured parameters.
func ExitErrParams(scope LogScope, err error, params map[string]any) {
	dur := time.Since(scope.Start)
	e := scope.Log.Error().
		Str(FieldEvent, "exit").
		Str(FieldResult, "error").
		Dur(FieldDuration, dur).
		Err(err)
	if params != nil {
		AddParams(e, zerolog.DebugLevel, params)
	}
	e.Msg("")
}

// ErrorContinue logs an error without closing the surrounding function scope.
func ErrorContinue(scope LogScope, err error, params map[string]any) {
	dur := time.Since(scope.Start)
	e := scope.Log.Error().
		Str(FieldEvent, "error").
		Dur(FieldDuration, dur).
		Err(err)
	if params != nil {
		AddParams(e, zerolog.WarnLevel, params)
	}
	e.Msg("")
}

func Debug(scope LogScope, event string, params map[string]any) {
	logg := scope.Log
	e := logg.Debug().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	e.Msg("")
}
func Trace(scope LogScope, event string, params map[string]any) {
	logg := scope.Log
	e := logg.Trace().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	e.Msg("")
}
func Info(scope LogScope, event string, params map[string]any) {
	logg := scope.Log
	e := logg.Info().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	e.Msg("")
}
func Warn(scope LogScope, event string, params map[string]any) {
	logg := scope.Log
	e := logg.Warn().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	e.Msg("")
}
func Fatal(scope LogScope, event string, params map[string]any, err error, msg string) {
	logg := scope.Log
	e := logg.Fatal().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	if err != nil {
		e.Err(err)
	}
	e.Msg(msg)
}
func Panic(scope LogScope, event string, params map[string]any, err error, msg string) {
	logg := scope.Log
	e := logg.Panic().
		Str(FieldEvent, event)
	if params != nil {
		AddParams(e, logg.GetLevel(), params)
	}
	if err != nil {
		e.Err(err)
	}
	e.Msg(msg)
}

// Return logs the function exit status using the appropriate exit helper
// and returns the provided error unchanged.
func Return(scope LogScope, err error) error {
	if err != nil {
		ExitErr(scope, err)
	} else {
		Exit(scope, "ok", nil)
	}
	return err
}

// Return logs the function exit status using the appropriate exit helper
// and returns the provided error unchanged.
func ReturnParams(scope LogScope, err error, params map[string]any) error {
	if err != nil {
		ExitErrParams(scope, err, params)
	} else {
		Exit(scope, "ok", params)
	}
	return err
}

//
// ===== Logger initialization =====
//

// LoadLogging replaces the global zerolog logger by loading
// a zeroconfig YAML configuration from the given file path.
//
// The function aborts the process if the configuration is invalid.
func LoadLogging(file string) {
	var f *os.File
	f, err := os.Open(file)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not readable")
		panic(err)
	}
	data, err := io.ReadAll(f)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not readable")
		panic(err)
	}
	var cfg LoggingConfig
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not valid yaml")
		panic(err)
	}
	logger, err := cfg.Compile()
	if err != nil {
		log.Logger.Fatal().Err(err).
			Msg(file + " is not valid for zerolog, see go.mau.fi/zeroconfig documentation")
		panic(err)
	}
	log.Logger = *logger

	Configure(cfg)

}
