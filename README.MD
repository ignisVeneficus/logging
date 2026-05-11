# logging

Structured, process-oriented logging helpers built on top of zerolog.

The package uses zeroconfig for logger configuration loading and compilation.

This package provides:

- Hierarchical scope-based logging
- Trace and span propagation through `context.Context`
- Structured enter/exit lifecycle logging
- Automatic duration measurement
- Configurable per-scope log levels
- Level-aware object marshaling
- Consistent machine-readable JSON logs

Designed for long-running pipelines, background workers, APIs, and highly concurrent Go applications.

## Features

- Structured JSON logging
- Trace/span correlation
- Function lifecycle logging (`enter` / `exit`)
- Context propagation across goroutines
- Hierarchical scope names (`sync/pipeline/worker`)
- Per-scope log level overrides
- Automatic execution timing
- Structured parameter logging
- Level-aware object serialization
- Thin wrapper around zerolog

# Installation

```bash
go get github.com/ignisVeneficus/logging
```


# Quick Example

```go
func ProcessImage(ctx context.Context, path string) error {
    logg, ctx := logging.Enter(
        ctx,
        "pipeline/image/process",
        path,
        map[string]any{
            "path": path,
        },
    )
    defer func() {
        logging.Exit(logg, "ok", nil)
    }()

    logging.Debug(logg, "metadata.read", map[string]any{
        "reader": "exiftool",
    })

    return nil
}
```

Example output:

```json
{
  "level":"debug",
  "trace_id":"39ab6400d3e49ebd7654f145469f759e",
  "parent_span_id":"",
  "scope":"pipeline/image/process",
  "span_id":"de78da6e600ae375",
  "anchor":"IMG_001.CR2",
  "event":"enter",
  "params":{
    "path":"IMG_001.CR2"
  }
}
```

```json
{
  "level":"debug",
  "trace_id":"39ab6400d3e49ebd7654f145469f759e",
  "parent_span_id":"",
  "scope":"pipeline/image/process",
  "span_id":"de78da6e600ae375",
  "anchor":"IMG_001.CR2",
  "event":"metadata.read",
  "params":{
    "reader":"exiftool"
  }
}
```

```json
{
  "level":"debug",
  "trace_id":"39ab6400d3e49ebd7654f145469f759e",
  "parent_span_id":"",
  "scope":"pipeline/image/process",
  "span_id":"de78da6e600ae375",
  "anchor":"IMG_001.CR2",
  "event":"exit",
  "duration":24.9532,
  "result":"ok"
}
```

# Core Concepts

## Trace ID

A trace represents a complete logical operation.

Example:

- HTTP request
- Sync run
- Background job
- Pipeline execution

All nested scopes inherit the same `trace_id`.

## Span ID

Each `Enter()` creates a new span.

Spans form a hierarchy through `parent_span_id`.

This makes it possible to reconstruct execution trees from logs.

## Parent Span ID

`parent_span_id` links spans together into a tree structure.

When a scope creates a child scope through `Enter()`, the child inherits:

- the same `trace_id`
- the current span as `parent_span_id`

This relationship makes it possible to reconstruct nested execution flows and concurrent processing trees from plain logs.

Example:

```text
pipeline/run                 span=a1 parent=
pipeline/run/read            span=b1 parent=a1
pipeline/run/read/metadata   span=c1 parent=b1
```

## Scope Names

Scopes are hierarchical paths:

```text
server/request
server/request/auth
pipeline/job/run
pipeline/job/run/metadata
```

Scopes are also used for log level configuration inheritance.

## Anchor

`anchor` is the human-readable counterpart of a `span_id`.

While `span_id` uniquely identifies a scope for machines, `anchor` identifies what the scope is currently processing.

Typical examples:

- image path
- filename
- request path
- job ID
- database entity ID

This makes traces significantly easier to inspect manually.

Example:

```text
scope: pipeline/job/process
span_id: de78da6e600ae375
anchor: IMG_2048.CR2
```

## Params

Structured contextual data is stored under the `params` field.

Parameters are intended for machine-readable execution metadata.

Typical examples:

- query parameters
- processing decisions
- worker state
- file metadata
- retry counts
- evaluation results

Example:

```json
{
  "event":"image.filtered",
  "params":{
    "reason":"rating",
    "rating":2
  }
}
```

Parameters are always structured objects, making them suitable for:

- jq filtering
- SQL ingestion
- log aggregation systems
- trace analysis tools


# Function Lifecycle Logging

The package is designed around explicit function scopes.

## Enter

Starts a scope and creates a child span.

```go
logg, ctx := logging.Enter(
    ctx,
    "sync/file/process",
    file.Path,
    map[string]any{
        "file": file.Path,
    },
)
```

Returns:

- `LogScope`
- derived `context.Context`


## Exit

Successful completion:

```go
logging.Exit(logg, "ok", nil)
```


## ExitErr

Error completion:

```go
logging.ExitErr(logg, err)
```


## Return Helper

Common shorthand pattern.

`logging.Return()` is equivalent to:

```go
if err != nil {
    logging.ExitErr(logg, err)
} else {
    logging.Exit(logg, "ok", nil)
}

return err
```

Example:

```go
func Run(ctx context.Context) error {
    logg, ctx := logging.Enter(ctx, "job/run", nil, nil)

    err := doWork(ctx)

    return logging.Return(logg, err)
}
```


# Logging Events

## Debug

```go
logging.Debug(logg, "cache.hit", map[string]any{
    "key": key,
})
```

## Info

```go
logging.Info(logg, "worker.started", nil)
```

## Warn

```go
logging.Warn(logg, "retry", map[string]any{
    "attempt": 2,
})
```

## Trace

```go
logging.Trace(logg, "sql.query", map[string]any{
    "query": query,
})
```

## ErrorContinue

Logs an error without closing the scope.

```go
logging.ErrorContinue(logg, err, map[string]any{
    "retry": true,
})
```


# Context Propagation

The logger context is stored in `context.Context`.

Derived contexts automatically inherit:

- trace ID
- current span ID
- parent span linkage

Example:

```go
func Worker(ctx context.Context) {
    logg, ctx := logging.Enter(ctx, "worker/run", nil, nil)

    process(ctx)
}
```


# Detached Background Contexts

`Detach()` creates a fresh background context while preserving logging trace information.

Useful for:

- queues
- background workers
- async processing

```go
bg := logging.Detach(ctx)

go worker(bg)
```


# Configuration

The package supports hierarchical log level configuration.

## YAML Example

```yaml
time_format: unixms

writers:
  - type: console

loggers:
  pipeline: debug
  pipeline/job: trace
  server: info
```

The logging backend configuration is provided by zeroconfig.

This package extends the zeroconfig YAML schema with an additional `loggers` section used for hierarchical scope-level overrides.

The `loggers` section is resolved using scope inheritance rules:

```yaml
loggers:
  pipeline: debug
  pipeline/job: trace
```

This allows fine-grained runtime control over individual subsystems without modifying application code.

See zeroconfig documentation for the base logger configuration format.

# Hierarchical Level Resolution

Scopes inherit levels from parent scopes.

Example:

```yaml
loggers:
  pipeline: debug
```

Then all of these inherit `debug`:

```text
pipeline/job
pipeline/job/run
pipeline/job/run/filter
```

More specific scopes override parent levels:

```yaml
loggers:
  pipeline: debug
  pipeline/job/run: trace
```


# Loading Configuration

```go
logging.LoadLogging("logging.yaml")
```

This:

- loads YAML
- configures zerolog
- installs the global logger
- configures scope log levels


# Structured Parameters

Parameters are stored under the `params` field.

```go
logging.Debug(logg, "image.filtered", map[string]any{
    "reason": "rating",
    "rating": 2,
})
```

Output:

```json
{
  "event":"image.filtered",
  "params":{
    "reason":"rating",
    "rating":2
  }
}
```

# Anchors

`anchor` is an optional high-level identifier attached to a scope.

Typical examples:

- filename
- image ID
- request path
- job ID

```go
logging.Enter(ctx, "sync/file", path, nil)
```


# Level-Aware Object Logging


Objects can customize their serialization depending on the active log level.

This functionality is fully standalone and can also be used directly with the original zerolog `Object()` APIs, independently from the tracing and scope logging features provided by this package.

Example standalone usage:

```go
d.Object(label, WithLevel(level, struct))
```

This allows level-aware object serialization in any zerolog event pipeline.


Implement:

```go
type ObjectWithLevel interface {
    MarshalZerologObjectWithLevel(
        e *zerolog.Event,
        level zerolog.Level,
    )
}
```

Example:

```go
type User struct {
    ID    uint64
    Email string
}

func (u User) MarshalZerologObjectWithLevel(
    e *zerolog.Event,
    level zerolog.Level,
) {
    e.Uint64("id", u.ID)

    if level <= zerolog.DebugLevel {
        e.Str("email", u.Email)
    }
}
```

## Conditional Field Helpers

Helpers for optional fields:

```go
logging.StrIf(e, "name", name)
logging.Int64If(e, "size", size)
logging.TimeIf(e, "created", createdAt)
```

Available helpers include:

- `IntIf`
- `Int16If`
- `Uint16If`
- `Int64If`
- `Uint64If`
- `BoolIf`
- `StrIf`
- `Float32If`
- `Float64If`
- `TimeIf`
- `ObjectIf`

These helpers are intended for writing compact, allocation-friendly structured object serializers without repetitive nil checks.


# Standard Log Fields

The package uses a consistent event schema:

| Field | Description |
|---|---|
| `event` | Event type |
| `result` | Result status |
| `scope` | Hierarchical scope name |
| `trace_id` | Trace identifier |
| `span_id` | Current span |
| `parent_span_id` | Parent span |
| `duration` | Execution duration |
| `params` | Structured parameters |
| `anchor` | Optional primary object |

# Intended Use Cases

- Long-running pipelines
- ETL systems
- Media processing
- Distributed workers
- Background jobs
- Request tracing
- Sync engines
- Batch processing
- Debuggable concurrent systems


# Design Goals

This package intentionally favors:

- explicit scopes over implicit magic
- machine-readable logs
- deterministic structure
- low allocation overhead
- grep/jq/sql-friendly output
- trace reconstruction from plain logs

It is not intended to replace:

- OpenTelemetry
- distributed tracing systems
- metrics systems
- APM platforms

Instead, it provides lightweight structured execution tracing directly inside application logs.

