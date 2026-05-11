package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/rs/zerolog"
)

func TestNewSpanIDHasExpectedLength(t *testing.T) {
	id := newSpanID()

	if len(id) != 16 {
		t.Fatalf("expected span id length 16, got %d: %q", len(id), id)
	}
}

func TestNewSpanIDGeneratesDifferentValues(t *testing.T) {
	id1 := newSpanID()
	id2 := newSpanID()

	if id1 == id2 {
		t.Fatalf("expected different span ids, got same value: %q", id1)
	}
}

func TestNewTraceIDHasExpectedLength(t *testing.T) {
	id := newTraceID()

	if len(id) != 32 {
		t.Fatalf("expected trace id length 32, got %d: %q", len(id), id)
	}
}

func TestNewTraceIDGeneratesDifferentValues(t *testing.T) {
	id1 := newTraceID()
	id2 := newTraceID()

	if id1 == id2 {
		t.Fatalf("expected different trace ids, got same value: %q", id1)
	}
}
func TestRootLogContextCreatesTraceID(t *testing.T) {
	lc := RootLogContext()

	if lc.TraceID == "" {
		t.Fatal("trace id is empty")
	}
}

func TestRootLogContextHasNoSpanID(t *testing.T) {
	lc := RootLogContext()

	if lc.SpanID != "" {
		t.Fatalf("expected empty span id, got %q", lc.SpanID)
	}
}

func TestRootLogContextCreatesDifferentTraceIDs(t *testing.T) {
	lc1 := RootLogContext()
	lc2 := RootLogContext()

	if lc1.TraceID == lc2.TraceID {
		t.Fatalf(
			"expected different trace ids, got same value: %q",
			lc1.TraceID,
		)
	}
}

func TestFromCtxNilCreatesFreshContext(t *testing.T) {
	lc := fromCtx(nil)

	if lc.TraceID == "" {
		t.Fatal("trace id is empty")
	}

	if lc.SpanID != "" {
		t.Fatalf("expected empty span id, got %q", lc.SpanID)
	}
}

func TestFromCtxMissingValueCreatesFreshContext(t *testing.T) {
	ctx := context.Background()

	lc := fromCtx(ctx)

	if lc.TraceID == "" {
		t.Fatal("trace id is empty")
	}

	if lc.SpanID != "" {
		t.Fatalf("expected empty span id, got %q", lc.SpanID)
	}
}

func TestFromCtxReturnsStoredContext(t *testing.T) {
	expected := LogContext{
		TraceID: "trace-1",
		SpanID:  "span-1",
	}

	ctx := context.WithValue(
		context.Background(),
		LogContextName,
		expected,
	)

	got := fromCtx(ctx)

	if got.TraceID != expected.TraceID {
		t.Fatalf(
			"trace id mismatch: expected=%q got=%q",
			expected.TraceID,
			got.TraceID,
		)
	}

	if got.SpanID != expected.SpanID {
		t.Fatalf(
			"span id mismatch: expected=%q got=%q",
			expected.SpanID,
			got.SpanID,
		)
	}
}

func assertLogContains(t *testing.T, buf *bytes.Buffer, expected map[string]any) {
	t.Helper()

	var got map[string]any

	err := json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	delete(got, "level")
	delete(got, "time")
	delete(got, "message")

	if !reflect.DeepEqual(expected, got) {
		t.Fatalf(
			"log mismatch\nexpected: %#v\ngot: %#v",
			expected,
			got,
		)
	}
}
func TestAddParamsAddsValues(t *testing.T) {
	var buf bytes.Buffer

	logger := zerolog.New(&buf)

	e := logger.Info()

	AddParams(e, zerolog.InfoLevel, map[string]any{
		"a": 123,
		"b": "test",
	})

	e.Msg("")

	assertLogContains(t, &buf, map[string]any{
		FieldParams: map[string]any{
			"a": float64(123),
			"b": "test",
		},
	})
}
