package logging

import (
	"time"

	"github.com/rs/zerolog"
)

//
// ===== Object-level logging =====
//

// ObjectWithLevel allows an object to control how it is marshaled
// depending on the active log level.
type ObjectWithLevel interface {
	MarshalZerologObjectWithLevel(e *zerolog.Event, level zerolog.Level)
}

// withLevel wraps an ObjectWithLevel with an explicit log level.
type withLevel struct {
	level zerolog.Level
	obj   ObjectWithLevel
}

// WithLevel wraps an ObjectWithLevel for level-aware marshaling.
func WithLevel(level zerolog.Level, obj ObjectWithLevel) *withLevel {
	if obj == nil {
		return nil
	}
	return &withLevel{level: level, obj: obj}
}

// MarshalZerologObject implements zerolog object marshaling.
func (w *withLevel) MarshalZerologObject(e *zerolog.Event) {
	w.obj.MarshalZerologObjectWithLevel(e, w.level)
}

//
// ===== Typed conditional field helpers =====
//

// IntIf logs an int value if not nil.
func IntIf(e *zerolog.Event, k string, v *int) {
	if v != nil {
		e.Int(k, *v)
	}
}

// Int16If logs an int16 value if not nil.
func Int16If(e *zerolog.Event, k string, v *int16) {
	if v != nil {
		e.Int16(k, *v)
	}
}

// Uint16If logs an uint16 value if not nil.
func Uint16If(e *zerolog.Event, k string, v *uint16) {
	if v != nil {
		e.Uint16(k, *v)
	}
}

// Int64If logs an int64 value if not nil.
func Int64If(e *zerolog.Event, k string, v *int64) {
	if v != nil {
		e.Int64(k, *v)
	}
}

// Uint64If logs an uint64 value if not nil.
func Uint64If(e *zerolog.Event, k string, v *uint64) {
	if v != nil {
		e.Uint64(k, *v)
	}
}

// BoolIf logs a bool value if not nil.
func BoolIf(e *zerolog.Event, k string, v *bool) {
	if v != nil {
		e.Bool(k, *v)
	}
}

// StrIf logs a string value if not nil.
func StrIf(e *zerolog.Event, k string, v *string) {
	if v != nil {
		e.Str(k, *v)
	}
}

// Float64If logs a float64 value if not nil.
func Float64If(e *zerolog.Event, k string, v *float64) {
	if v != nil {
		e.Float64(k, *v)
	}
}

// Float32If logs a float32 value if not nil.
func Float32If(e *zerolog.Event, k string, v *float32) {
	if v != nil {
		e.Float32(k, *v)
	}
}

// TimeIf logs a time.Time value if not nil.
func TimeIf(e *zerolog.Event, k string, v *time.Time) {
	if v != nil {
		e.Time(k, *v)
	}
}

// ObjectIf logs a structured object if present.
// If logNil is true, a nil value is explicitly logged.
func ObjectIf(e *zerolog.Event, key string, w *withLevel, logNil bool) {
	if w == nil {
		if logNil {
			e.Interface(key, nil)
		}
		return
	}
	e.Object(key, w)
}
