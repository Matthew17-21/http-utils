package httputils

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// captureOutput captures stdout during test execution
func captureOutput(fn func()) string {
	// Save original stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute the function
	fn()

	// Restore stdout
	w.Close()
	os.Stdout = old

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestNewLogger(t *testing.T) {
	t.Run("creates logger instance", func(t *testing.T) {
		logger := NewLogger()
		require.NotNil(t, logger)
		require.Implements(t, (*Logger)(nil), logger)
	})

	t.Run("logger implements Logger interface", func(t *testing.T) {
		var _ Logger = NewLogger()
	})
}

func TestLoggerMethods(t *testing.T) {
	tests := []struct {
		name     string
		logFunc  func(Logger)
		expected string
	}{
		{
			name: "Debug method",
			logFunc: func(l Logger) {
				l.Debug("debug message %s", "test")
			},
			expected: "", // Debug should not output anything
		},
		{
			name: "Info method",
			logFunc: func(l Logger) {
				l.Info("info message %s", "test")
			},
			expected: "info message test\n",
		},
		{
			name: "Warn method",
			logFunc: func(l Logger) {
				l.Warn("warn message %s", "test")
			},
			expected: "warn message test\n",
		},
		{
			name: "Error method",
			logFunc: func(l Logger) {
				l.Error("error message %s", "test")
			},
			expected: "error message test\n",
		},
		{
			name: "Info with multiple arguments",
			logFunc: func(l Logger) {
				l.Info("multiple args: %s, %d, %v", "string", 42, true)
			},
			expected: "multiple args: string, 42, true\n",
		},
		{
			name: "Info with no arguments",
			logFunc: func(l Logger) {
				l.Info("simple message")
			},
			expected: "simple message\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := NewLogger()
			output := captureOutput(func() {
				test.logFunc(logger)
			})
			require.Equal(t, test.expected, output)
		})
	}
}

func TestNewSilentLogger(t *testing.T) {
	t.Run("creates silent logger instance", func(t *testing.T) {
		logger := NewSilentLogger()
		require.NotNil(t, logger)
		require.Implements(t, (*Logger)(nil), logger)
	})

	t.Run("silent logger implements Logger interface", func(t *testing.T) {
		var _ Logger = NewSilentLogger()
	})
}

func TestSilentLoggerMethods(t *testing.T) {
	tests := []struct {
		name    string
		logFunc func(Logger)
	}{
		{
			name: "Debug method",
			logFunc: func(l Logger) {
				l.Debug("debug message %s", "test")
			},
		},
		{
			name: "Info method",
			logFunc: func(l Logger) {
				l.Info("info message %s", "test")
			},
		},
		{
			name: "Warn method",
			logFunc: func(l Logger) {
				l.Warn("warn message %s", "test")
			},
		},
		{
			name: "Error method",
			logFunc: func(l Logger) {
				l.Error("error message %s", "test")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := NewSilentLogger()
			output := captureOutput(func() {
				test.logFunc(logger)
			})
			require.Empty(t, output, "Silent logger should not output anything")
		})
	}
}

func TestNewDebugLogger(t *testing.T) {
	t.Run("creates debug logger instance", func(t *testing.T) {
		baseLogger := NewLogger()
		debugLogger := NewDebugLogger(baseLogger)
		require.NotNil(t, debugLogger)
		require.Implements(t, (*Logger)(nil), debugLogger)
	})

	t.Run("debug logger implements Logger interface", func(t *testing.T) {
		baseLogger := NewLogger()
		var _ Logger = NewDebugLogger(baseLogger)
	})

	t.Run("creates debug logger with silent base", func(t *testing.T) {
		baseLogger := NewSilentLogger()
		debugLogger := NewDebugLogger(baseLogger)
		require.NotNil(t, debugLogger)
	})
}

func TestDebugLoggerMethods(t *testing.T) {
	t.Run("Debug method outputs to stdout", func(t *testing.T) {
		baseLogger := NewSilentLogger()
		debugLogger := NewDebugLogger(baseLogger)

		output := captureOutput(func() {
			debugLogger.Debug("debug message %s", "test")
		})
		require.Equal(t, "debug message test\n", output)
	})

	t.Run("Info method delegates to base logger", func(t *testing.T) {
		baseLogger := NewLogger()
		debugLogger := NewDebugLogger(baseLogger)

		output := captureOutput(func() {
			debugLogger.Info("info message %s", "test")
		})
		require.Equal(t, "info message test\n", output)
	})

	t.Run("Warn method delegates to base logger", func(t *testing.T) {
		baseLogger := NewLogger()
		debugLogger := NewDebugLogger(baseLogger)

		output := captureOutput(func() {
			debugLogger.Warn("warn message %s", "test")
		})
		require.Equal(t, "warn message test\n", output)
	})

	t.Run("Error method delegates to base logger", func(t *testing.T) {
		baseLogger := NewLogger()
		debugLogger := NewDebugLogger(baseLogger)

		output := captureOutput(func() {
			debugLogger.Error("error message %s", "test")
		})
		require.Equal(t, "error message test\n", output)
	})

	t.Run("with silent base logger", func(t *testing.T) {
		baseLogger := NewSilentLogger()
		debugLogger := NewDebugLogger(baseLogger)

		// Debug should still output
		debugOutput := captureOutput(func() {
			debugLogger.Debug("debug message")
		})
		require.Equal(t, "debug message\n", debugOutput)

		// Other methods should be silent
		otherOutput := captureOutput(func() {
			debugLogger.Info("info message")
			debugLogger.Warn("warn message")
			debugLogger.Error("error message")
		})
		require.Empty(t, otherOutput)
	})
}

func TestLoggerEdgeCases(t *testing.T) {
	t.Run("logger with empty format string", func(t *testing.T) {
		logger := NewLogger()
		output := captureOutput(func() {
			logger.Info("")
		})
		require.Equal(t, "\n", output)
	})

	t.Run("logger with nil arguments", func(t *testing.T) {
		logger := NewLogger()
		output := captureOutput(func() {
			logger.Info("message with nil: %v", nil)
		})
		require.Equal(t, "message with nil: <nil>\n", output)
	})

	t.Run("logger with special characters", func(t *testing.T) {
		logger := NewLogger()
		output := captureOutput(func() {
			logger.Info("special chars: %s", "!@#$%^&*()")
		})
		require.Equal(t, "special chars: !@#$%^&*()\n", output)
	})

	t.Run("logger with unicode characters", func(t *testing.T) {
		logger := NewLogger()
		output := captureOutput(func() {
			logger.Info("unicode: %s", "🚀🌟✨")
		})
		require.Equal(t, "unicode: 🚀🌟✨\n", output)
	})

	t.Run("logger with very long message", func(t *testing.T) {
		logger := NewLogger()
		longMessage := strings.Repeat("a", 1000)
		output := captureOutput(func() {
			logger.Info("long: %s", longMessage)
		})
		expected := fmt.Sprintf("long: %s\n", longMessage)
		require.Equal(t, expected, output)
	})
}

func TestLoggerConcurrency(t *testing.T) {
	t.Run("multiple goroutines logging simultaneously", func(t *testing.T) {
		logger := NewLogger()
		done := make(chan bool, 10)

		output := captureOutput(func() {
			for i := 0; i < 10; i++ {
				go func(id int) {
					logger.Info("goroutine %d", id)
					done <- true
				}(i)
			}

			// Wait for all goroutines to complete
			for i := 0; i < 10; i++ {
				<-done
			}
		})

		// Should have 10 lines of output
		lines := strings.Split(strings.TrimSpace(output), "\n")
		require.Len(t, lines, 10)

		// Each line should contain a goroutine ID
		for i := 0; i < 10; i++ {
			require.Contains(t, output, fmt.Sprintf("goroutine %d", i))
		}
	})
}

func TestLoggerComposition(t *testing.T) {
	t.Run("debug logger with debug logger", func(t *testing.T) {
		baseLogger := NewLogger()
		debugLogger1 := NewDebugLogger(baseLogger)
		debugLogger2 := NewDebugLogger(debugLogger1)

		output := captureOutput(func() {
			debugLogger2.Debug("debug message")
			debugLogger2.Info("info message")
		})

		// Should have both debug and info output
		require.Contains(t, output, "debug message")
		require.Contains(t, output, "info message")
		require.Equal(t, 2, strings.Count(output, "\n"))
	})
}

func BenchmarkLogger(b *testing.B) {
	logger := NewLogger()

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message %d", i)
		}
	})

	b.Run("Debug", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Debug("benchmark message %d", i)
		}
	})
}

func BenchmarkSilentLogger(b *testing.B) {
	logger := NewSilentLogger()

	b.Run("Info", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message %d", i)
		}
	})
}

func BenchmarkDebugLogger(b *testing.B) {
	baseLogger := NewLogger()
	debugLogger := NewDebugLogger(baseLogger)

	b.Run("Debug", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			debugLogger.Debug("benchmark message %d", i)
		}
	})
}
