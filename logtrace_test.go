package logtrace

import (
	"bytes"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/ansel1/merry/v2"
	goerr "github.com/go-errors/errors"
	"github.com/shoenig/test"
	"github.com/sirupsen/logrus"
)

const TEST_STACKTRACE_KEY = "test_stacktrace"

func testHook(capture bool) LogtraceHook {
	return LogtraceHook{
		Key:                   TEST_STACKTRACE_KEY,
		LogLevels:             logrus.AllLevels,
		CaptureStackIfMissing: capture,
	}
}

func brokenTestHook() LogtraceHook {
	return LogtraceHook{
		Key:                   "",
		LogLevels:             logrus.AllLevels,
		CaptureStackIfMissing: false,
	}
}

type voidWriter struct{}

func (voidWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

type validationFormatter struct {
	Data *logrus.Fields
}

func (vF validationFormatter) Format(e *logrus.Entry) ([]byte, error) {
	*vF.Data = e.Data
	return []byte{}, nil
}

func TestDefaultLogtraceHook(t *testing.T) {
	hook := DefaultLogtraceHook()
	test.Eq(t, "stacktrace", hook.Key)
	test.Eq(t, logrus.AllLevels, hook.LogLevels)
	test.True(t, hook.CaptureStackIfMissing)
}

func setupErrorTracebackHookTest() (validationFormatter, *logrus.Logger) {
	formatter := validationFormatter{
		Data: new(logrus.Fields),
	}
	testLogger := &logrus.Logger{
		Out:       voidWriter{},
		Formatter: formatter,
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.DebugLevel,
	}
	return formatter, testLogger
}

func captureStderr(f func()) (error, string) {
	orig := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	var buf bytes.Buffer
	w.Close()
	os.Stderr = orig

	_, err := io.Copy(&buf, r)

	return err, buf.String()
}

func TestErrorTracebackHook(t *testing.T) {
	testCases := []struct {
		name                 string
		hook                 LogtraceHook
		err                  error
		shouldHaveStacktrace bool
		expectedStderr       string
	}{
		{"builtin error without CaptureStacktraceIfMissing", testHook(false), errors.New("Testerror"), false, ""},
		{"builtin error with CaptureStacktraceIfMissing", testHook(true), errors.New("Testerror"), true, ""},
		{"goerror error", testHook(false), goerr.New("Testerror"), true, ""},
		{"merry error", testHook(false), merry.New("Testerror"), true, ""},
		{"brokenTestHook", brokenTestHook(), merry.New("Testerror"), false, "Failed to fire hook: The key of the hook must be set to a non empty string.\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatter, testLogger := setupErrorTracebackHookTest()
			testLogger.AddHook(tc.hook)

			err, stderr := captureStderr(func() { testLogger.WithError(tc.err).Info("") })
			if err != nil {
				t.Fatal("Failed to capture stderr. Please run test again.")
			}
			test.Eq(t, tc.expectedStderr, stderr)

			if val, ok := (*formatter.Data)[logrus.ErrorKey]; ok {
				test.True(t, val == tc.err)
			} else {
				test.Unreachable(t)
			}

			_, ok := (*formatter.Data)[TEST_STACKTRACE_KEY]
			test.Eq(t, tc.shouldHaveStacktrace, ok)
		})
	}
}

func TestErrorTracebackHookNonErrorError(t *testing.T) {
	_, testLogger := setupErrorTracebackHookTest()
	testLogger.AddHook(testHook(false))

	err, stderr := captureStderr(func() { testLogger.WithField(logrus.ErrorKey, "foo").Info("") })
	if err != nil {
		t.Fatal("Failed to capture stderr. Please run test again.")
	}
	test.Eq(t, "Failed to fire hook: The value in the error field of the given logrus entry is not an error!\n", stderr)
}
