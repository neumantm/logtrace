# logtrace

[![Go Reference](https://pkg.go.dev/badge/github.com/neumantm/logtrace.svg)](https://pkg.go.dev/github.com/neumantm/logtrace)
[![CI](https://github.com/neumantm/logtrace/actions/workflows/ci.yml/badge.svg)](https://github.com/neumantm/logtrace/actions/workflows/ci.yml)

Logrus hook for attaching an error traceback to a log entry

This project is a hook for the golang logging libary [logrus](https://github.com/sirupsen/logrus).
It attaches a stacktrace field to log entries with an error.
It is primarily developed to work with the error libary [merry/v2](https://github.com/ansel1/merry/tree/main/v2).
However, it also works with all errors implementing a `Callers()` function like for example the errors from [go-errors](https://github.com/go-errors/errors).

## Usage

To use the hook, just register it with logrus:

```go
import "github.com/neumantm/logtrace"

func setupLogging() {
    // ...
    logrus.AddHook(logtrace.DefaultLogtraceHook())
    // Alternatively with custom parameters:
    //logrus.AddHook(logtrace.LogtraceHook{
    //    Key:                   "SomeKey",
    //    LogLevels:             logrus.AllLevels,
    //    CaptureStackIfMissing: false,
    //})
    // ...
}
```

The default hook runs on all levels uses the key `stacktrace` and captures the stack if the error does not have a compatible stack.

## Stacktrace accuracy

Sometimes the entries on the stacktrace may not be what you expect because of compiler optimizations.
To avoid this problem (during development or debugging) disable compiler optimizations and inlining by passing `-gcflags '-N -l'` to `go build` or `go run`.
