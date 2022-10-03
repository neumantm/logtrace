package logtrace

import (
	"errors"
	"runtime"
	"strings"

	"github.com/ansel1/merry/v2"
	"github.com/sirupsen/logrus"
)

// LogtraceHook is a logrus hook that attaches a stacktrace to log entries with errors
type LogtraceHook struct {
	// The key (for the field in the logrus entry) to use for the stacktrace
	Key string
	// The log levels this hook should be fired for
	LogLevels []logrus.Level
	// Whether to capture a new stacktrace for errors without a stacktrace
	//
	// The captures tacktrace corresponds to the call of the log function, not the actual origin of the error
	// However, this may still help in identifying the path
	CaptureStackIfMissing bool
}

// DefaultLogtraceHook returns the default LogtraceHook
//
// It has the following properties
// - The key is "stacktrace"
// - The hook runs on all log levels
// - For errors without a stacktrace a new stacktrace is captured corresponding to the call of the log function
func DefaultLogtraceHook() LogtraceHook {
	return LogtraceHook{
		Key:                   "stacktrace",
		LogLevels:             logrus.AllLevels,
		CaptureStackIfMissing: true,
	}
}

func (hook LogtraceHook) Levels() []logrus.Level {
	return hook.LogLevels
}

// A StackFrame represents one frame in a stacktrace
type StackFrame struct {
	// File is the path to the file containing the instruction of the StackFrame
	File string
	// The Line of the instruction of the StackFrame
	Line int
	// The Function of the instruction of the StackFrame
	Function string
	// The Package that contains the Function
	Package string
}

// newStackFrame returns the StackFrame for the given programCounter
//
// Returns empty StackFrame if no function can be associated with the programCounter
//
// Based on and partially copied from github.com/go-errors/errors
// Copyright (c) 2015 Conrad Irwin <conrad@bugsnag.com>
// Licensed under MIT license
func newStackFrame(programCounter uintptr) StackFrame {
	f := runtime.FuncForPC(programCounter)
	if f == nil {
		return StackFrame{}
	}

	// pc -1 because the program counters we use are usually return addresses,
	// and we want to show the line that corresponds to the function call
	file, line := f.FileLine(programCounter - 1)

	name := f.Name()
	pkg := ""

	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//  runtime/debug.*T·ptrmethod
	// and want
	//  *T.ptrmethod
	// Since the package path might contains dots (e.g. code.google.com/...),
	// we first remove the path prefix if there is one.
	if lastslash := strings.LastIndex(name, "/"); lastslash >= 0 {
		pkg += name[:lastslash] + "/"
		name = name[lastslash+1:]
	}
	if period := strings.Index(name, "."); period >= 0 {
		pkg += name[:period]
		name = name[period+1:]
	}

	name = strings.Replace(name, "·", ".", -1)

	return StackFrame{
		File:     file,
		Line:     line,
		Function: name,
		Package:  pkg,
	}
}

func (hook LogtraceHook) retrieveProgramCounters(err error) []uintptr {
	var errorWithCallers interface {
		Callers() []uintptr
	}
	stack := merry.Stack(err)
	if stack != nil {
		// merry error
		return stack
	} else if errors.As(err, &errorWithCallers) {
		// go-errors style error
		return errorWithCallers.Callers()
	} else if hook.CaptureStackIfMissing {
		// incompatible error, capture stack now
		return merry.Stack(merry.WrapSkipping(err, 6))
	} else {
		// incompatible error, no stack
		return nil
	}
}

func (hook LogtraceHook) retrieveStackFrames(err error) []StackFrame {
	pcs := hook.retrieveProgramCounters(err)

	if pcs == nil || len(pcs) == 0 {
		return []StackFrame{}
	}

	result := make([]StackFrame, len(pcs))
	for idx, pc := range pcs {
		result[idx] = newStackFrame(pc)
	}
	return result
}

// Fire is called by logrus when to hook is supposed to run
func (hook LogtraceHook) Fire(entry *logrus.Entry) error {
	if hook.Key == "" {
		return merry.Errorf("The key of the hook must be set to a non empty string.")
	}
	if err_raw, hasError := entry.Data[logrus.ErrorKey]; hasError {
		err, ok := err_raw.(error)
		if !ok {
			return merry.Errorf("The value in the error field of the given logrus entry is not an error!")
		}
		stack := hook.retrieveStackFrames(err)
		if len(stack) > 0 {
			entry.Data = entry.WithField(hook.Key, stack).Data
		}
	}
	return nil
}
