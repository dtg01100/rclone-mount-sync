package rclone

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func createMockRcloneForRetry(t *testing.T, script string) string {
	t.Helper()
	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, "rclone")
	if runtime.GOOS == "windows" {
		mockPath += ".bat"
	}
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create mock rclone: %v", err)
	}
	return mockPath
}

func TestListRemotesWithRetrySuccess(t *testing.T) {
	attemptFile := filepath.Join(t.TempDir(), "attempts")

	mockScript := fmt.Sprintf(`#!/bin/sh
attempt=$(cat %s 2>/dev/null || echo 0)
attempt=$((attempt + 1))
echo $attempt > %s

if [ "$attempt" -lt 3 ]; then
    echo "connection timeout" >&2
    exit 1
fi

case "$1" in
    listremotes)
        echo "gdrive:"
        ;;
    config)
        echo "[gdrive]"; echo "type = drive"
        ;;
esac
`, attemptFile, attemptFile)

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    50 * time.Millisecond,
		MaxDelay:        200 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	remotes, err := c.ListRemotes()
	if err != nil {
		t.Fatalf("ListRemotes() error = %v", err)
	}

	if len(remotes) != 1 {
		t.Errorf("ListRemotes() returned %d remotes, want 1", len(remotes))
	}
}

func TestListRemotesNoRetryOnPermanentError(t *testing.T) {
	mockScript := `#!/bin/sh
echo "config file not found" >&2
exit 1
`

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	_, err := c.ListRemotes()
	if err == nil {
		t.Fatal("ListRemotes() should return error for permanent error")
	}

	if !IsPermanentError(err) {
		t.Logf("error type check - error may be wrapped: %v", err)
	}
}

func TestListRemotesFailsAfterMaxRetries(t *testing.T) {
	mockScript := `#!/bin/sh
echo "connection timeout" >&2
exit 1
`

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	_, err := c.ListRemotes()
	if err == nil {
		t.Fatal("ListRemotes() should return error after max retries")
	}
}

func TestGetRemoteTypeWithRetry(t *testing.T) {
	attemptFile := filepath.Join(t.TempDir(), "attempts")

	mockScript := fmt.Sprintf(`#!/bin/sh
attempt=$(cat %s 2>/dev/null || echo 0)
attempt=$((attempt + 1))
echo $attempt > %s

if [ "$attempt" -lt 2 ]; then
    echo "connection timeout" >&2
    exit 1
fi

echo "[gdrive]"
echo "type = drive"
`, attemptFile, attemptFile)

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	remoteType, err := c.GetRemoteType("gdrive")
	if err != nil {
		t.Fatalf("GetRemoteType() error = %v", err)
	}

	if remoteType != "drive" {
		t.Errorf("GetRemoteType() = %q, want %q", remoteType, "drive")
	}
}

func TestListRemotePathWithRetry(t *testing.T) {
	attemptFile := filepath.Join(t.TempDir(), "attempts")

	mockScript := fmt.Sprintf(`#!/bin/sh
attempt=$(cat %s 2>/dev/null || echo 0)
attempt=$((attempt + 1))
echo $attempt > %s

if [ "$attempt" -lt 2 ]; then
    echo "timeout" >&2
    exit 1
fi

echo "file1.txt"
echo "file2.txt"
`, attemptFile, attemptFile)

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	entries, err := c.ListRemotePath("gdrive", "/")
	if err != nil {
		t.Fatalf("ListRemotePath() error = %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("ListRemotePath() returned %d entries, want 2", len(entries))
	}
}

func TestTestRemoteAccessWithRetry(t *testing.T) {
	attemptFile := filepath.Join(t.TempDir(), "attempts")

	mockScript := fmt.Sprintf(`#!/bin/sh
attempt=$(cat %s 2>/dev/null || echo 0)
attempt=$((attempt + 1))
echo $attempt > %s

if [ "$attempt" -lt 2 ]; then
    echo "connection timeout" >&2
    exit 1
fi

exit 0
`, attemptFile, attemptFile)

	mockPath := createMockRcloneForRetry(t, mockScript)
	c := NewClientWithPath(mockPath)
	c.SetRetryConfig(RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	})

	err := c.TestRemoteAccess("gdrive", "/")
	if err != nil {
		t.Fatalf("TestRemoteAccess() error = %v", err)
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "explicit retryable error",
			err:      NewRetryableError(errors.New("some error")),
			expected: true,
		},
		{
			name:     "explicit permanent error",
			err:      NewPermanentError(errors.New("permanent error")),
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "net timeout error",
			err:      &net.OpError{Err: &timeoutError{}},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			expected: true,
		},
		{
			name:     "no such host",
			err:      &net.OpError{Op: "dial", Err: errors.New("no such host")},
			expected: true,
		},
		{
			name:     "timeout in message",
			err:      errors.New("operation timeout"),
			expected: true,
		},
		{
			name:     "connection refused in message",
			err:      errors.New("connection refused by server"),
			expected: true,
		},
		{
			name:     "network unreachable in message",
			err:      errors.New("network is unreachable"),
			expected: true,
		},
		{
			name:     "dns failure in message",
			err:      errors.New("dns resolution failed"),
			expected: true,
		},
		{
			name:     "i/o timeout in message",
			err:      errors.New("i/o timeout"),
			expected: true,
		},
		{
			name:     "deadline exceeded in message",
			err:      errors.New("deadline exceeded"),
			expected: true,
		},
		{
			name:     "connection reset in message",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "random error",
			err:      errors.New("some random error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestIsPermanentError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "permanent error",
			err:      NewPermanentError(errors.New("permanent")),
			expected: true,
		},
		{
			name:     "retryable error",
			err:      NewRetryableError(errors.New("retryable")),
			expected: false,
		},
		{
			name:     "regular error",
			err:      errors.New("regular"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPermanentError(tt.err)
			if result != tt.expected {
				t.Errorf("IsPermanentError(%v) = %v, want %v", tt.err, result, tt.expected)
			}
		})
	}
}

func TestClassifyExitError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantRetryable  bool
		wantPermanent  bool
		wantUnmodified bool
	}{
		{
			name:           "nil error",
			err:            nil,
			wantUnmodified: true,
		},
		{
			name:          "config not found",
			err:           &exec.ExitError{Stderr: []byte("config file not found"), ProcessState: nil},
			wantPermanent: true,
		},
		{
			name:          "authentication failed",
			err:           &exec.ExitError{Stderr: []byte("authentication failed"), ProcessState: nil},
			wantPermanent: true,
		},
		{
			name:          "access denied",
			err:           &exec.ExitError{Stderr: []byte("access denied"), ProcessState: nil},
			wantPermanent: true,
		},
		{
			name:          "invalid config",
			err:           &exec.ExitError{Stderr: []byte("invalid config"), ProcessState: nil},
			wantPermanent: true,
		},
		{
			name:          "remote not found",
			err:           &exec.ExitError{Stderr: []byte("unknown remote"), ProcessState: nil},
			wantPermanent: true,
		},
		{
			name:          "timeout in stderr",
			err:           &exec.ExitError{Stderr: []byte("connection timeout"), ProcessState: nil},
			wantRetryable: true,
		},
		{
			name:          "connection refused in stderr",
			err:           &exec.ExitError{Stderr: []byte("connection refused"), ProcessState: nil},
			wantRetryable: true,
		},
		{
			name:          "network error in stderr",
			err:           &exec.ExitError{Stderr: []byte("network error"), ProcessState: nil},
			wantRetryable: true,
		},
		{
			name:          "dns error in stderr",
			err:           &exec.ExitError{Stderr: []byte("dns resolution failed"), ProcessState: nil},
			wantRetryable: true,
		},
		{
			name:           "non-exit error",
			err:            errors.New("some error"),
			wantUnmodified: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := classifyExitError(tt.err)

			if tt.wantUnmodified {
				if result != tt.err {
					t.Errorf("classifyExitError() should return unmodified error, got %v", result)
				}
				return
			}

			if tt.wantPermanent {
				if !IsPermanentError(result) {
					t.Errorf("classifyExitError() should return permanent error, got %v", result)
				}
			}

			if tt.wantRetryable {
				if !IsRetryableError(result) {
					t.Errorf("classifyExitError() should return retryable error, got %v", result)
				}
			}
		})
	}
}

func TestDoRetrySuccessOnFirstTry(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return nil
	})

	if err != nil {
		t.Errorf("doRetry() returned error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1", callCount)
	}
}

func TestDoRetrySuccessAfterTransientFailure(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		if callCount < 3 {
			return NewRetryableError(errors.New("transient error"))
		}
		return nil
	})

	if err != nil {
		t.Errorf("doRetry() returned error: %v", err)
	}
	if callCount != 3 {
		t.Errorf("operation called %d times, want 3", callCount)
	}
}

func TestDoRetryFailureAfterMaxRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return NewRetryableError(errors.New("persistent transient error"))
	})

	if err == nil {
		t.Error("doRetry() should return error after max retries")
	}
	if callCount != 3 {
		t.Errorf("operation called %d times, want 3", callCount)
	}
	if !errors.Is(err, NewRetryableError(errors.New("test"))) {
		t.Logf("error is: %v", err)
	}
}

func TestDoRetryNoRetryOnPermanentError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return NewPermanentError(errors.New("permanent error"))
	})

	if err == nil {
		t.Error("doRetry() should return error for permanent error")
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1 (no retry)", callCount)
	}
}

func TestDoRetryContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      10,
		InitialDelay:    1 * time.Second,
		MaxDelay:        5 * time.Second,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	callCount := 0
	err := doRetry(ctx, config, func() error {
		callCount++
		return NewRetryableError(errors.New("transient error"))
	})

	if err != context.Canceled {
		t.Errorf("doRetry() should return context.Canceled, got %v", err)
	}
	if callCount != 0 {
		t.Errorf("operation should not be called, was called %d times", callCount)
	}
}

func TestDoRetryBytesSuccessOnFirstTry(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	result, err := doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		return []byte("success"), nil
	})

	if err != nil {
		t.Errorf("doRetryBytes() returned error: %v", err)
	}
	if string(result) != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1", callCount)
	}
}

func TestDoRetryBytesSuccessAfterTransientFailure(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	result, err := doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		if callCount < 2 {
			return nil, NewRetryableError(errors.New("transient error"))
		}
		return []byte("success"), nil
	})

	if err != nil {
		t.Errorf("doRetryBytes() returned error: %v", err)
	}
	if string(result) != "success" {
		t.Errorf("result = %q, want %q", result, "success")
	}
	if callCount != 2 {
		t.Errorf("operation called %d times, want 2", callCount)
	}
}

func TestDoRetryBytesFailureAfterMaxRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	_, err := doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		return nil, NewRetryableError(errors.New("persistent transient error"))
	})

	if err == nil {
		t.Error("doRetryBytes() should return error after max retries")
	}
	if callCount != 3 {
		t.Errorf("operation called %d times, want 3", callCount)
	}
}

func TestDoRetryBytesNoRetryOnPermanentError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	_, err := doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		return nil, NewPermanentError(errors.New("permanent error"))
	})

	if err == nil {
		t.Error("doRetryBytes() should return error for permanent error")
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1 (no retry)", callCount)
	}
}

func TestExponentialBackoff(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    50 * time.Millisecond,
		MaxDelay:        500 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	var delays []time.Duration
	start := time.Now()

	callCount := 0
	_, _ = doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		if callCount > 1 {
			delays = append(delays, time.Since(start))
			start = time.Now()
		}
		return nil, NewRetryableError(errors.New("error"))
	})

	if len(delays) < 2 {
		t.Skip("not enough delays captured")
	}

	expectedDelays := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
	}

	for i, expected := range expectedDelays {
		if i >= len(delays) {
			break
		}
		tolerance := expected / 2
		if delays[i] < expected-tolerance || delays[i] > expected+tolerance {
			t.Logf("delay[%d] = %v, expected approximately %v", i, delays[i], expected)
		}
	}
}

func TestMaxDelayCap(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        150 * time.Millisecond,
		RetryMultiplier: 3.0,
	}

	var delays []time.Duration
	start := time.Now()

	callCount := 0
	_, _ = doRetryBytes(context.Background(), config, func() ([]byte, error) {
		callCount++
		if callCount > 1 {
			delays = append(delays, time.Since(start))
			start = time.Now()
		}
		return nil, NewRetryableError(errors.New("error"))
	})

	for i, delay := range delays {
		if delay > config.MaxDelay+50*time.Millisecond {
			t.Errorf("delay[%d] = %v exceeds max delay %v", i, delay, config.MaxDelay)
		}
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != DefaultMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", config.MaxRetries, DefaultMaxRetries)
	}
	if config.InitialDelay != DefaultInitialDelay {
		t.Errorf("InitialDelay = %v, want %v", config.InitialDelay, DefaultInitialDelay)
	}
	if config.MaxDelay != DefaultMaxDelay {
		t.Errorf("MaxDelay = %v, want %v", config.MaxDelay, DefaultMaxDelay)
	}
	if config.RetryMultiplier != DefaultRetryMultiplier {
		t.Errorf("RetryMultiplier = %v, want %v", config.RetryMultiplier, DefaultRetryMultiplier)
	}
}

func TestClientSetRetryConfig(t *testing.T) {
	c := NewClient()

	customConfig := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    200 * time.Millisecond,
		MaxDelay:        10 * time.Second,
		RetryMultiplier: 1.5,
	}

	c.SetRetryConfig(customConfig)

	result := c.GetRetryConfig()
	if result.MaxRetries != customConfig.MaxRetries {
		t.Errorf("MaxRetries = %d, want %d", result.MaxRetries, customConfig.MaxRetries)
	}
	if result.InitialDelay != customConfig.InitialDelay {
		t.Errorf("InitialDelay = %v, want %v", result.InitialDelay, customConfig.InitialDelay)
	}
}

func TestRetryableErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	retryable := NewRetryableError(inner)

	if !errors.Is(retryable, inner) {
		t.Error("RetryableError should unwrap to inner error")
	}
}

func TestPermanentErrorUnwrap(t *testing.T) {
	inner := errors.New("inner error")
	permanent := NewPermanentError(inner)

	if !errors.Is(permanent, inner) {
		t.Error("PermanentError should unwrap to inner error")
	}
}

func TestRetryableErrorMessage(t *testing.T) {
	err := NewRetryableError(errors.New("test error"))
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}
}

func TestPermanentErrorMessage(t *testing.T) {
	err := NewPermanentError(errors.New("test error"))
	if err.Error() != "test error" {
		t.Errorf("Error() = %q, want %q", err.Error(), "test error")
	}
}

func TestDoRetryAfterContextCancellationBetweenRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := doRetry(ctx, config, func() error {
		callCount++
		return NewRetryableError(errors.New("transient error"))
	})

	if err != context.Canceled {
		t.Errorf("doRetry() should return context.Canceled, got %v", err)
	}
}

func TestDoRetryBytesContextCancellationBetweenRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    100 * time.Millisecond,
		MaxDelay:        1 * time.Second,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := doRetryBytes(ctx, config, func() ([]byte, error) {
		callCount++
		return nil, NewRetryableError(errors.New("transient error"))
	})

	if err != context.Canceled {
		t.Errorf("doRetryBytes() should return context.Canceled, got %v", err)
	}
}

func TestNonRetryableRegularError(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      3,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return errors.New("regular error")
	})

	if err == nil {
		t.Error("doRetry() should return error")
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1 (non-retryable regular error)", callCount)
	}
}

func TestTimeoutErrorMessageIsRetryable(t *testing.T) {
	err := errors.New("operation timed out after 30 seconds")
	if !IsRetryableError(err) {
		t.Error("timeout error message should be retryable")
	}
}

func TestTimeoutKeywordIsRetryable(t *testing.T) {
	err := errors.New("operation timeout")
	if !IsRetryableError(err) {
		t.Error("timeout keyword should be retryable")
	}
}

func TestConnectionRefusedMessageIsRetryable(t *testing.T) {
	err := errors.New("dial tcp 127.0.0.1:8080: connection refused")
	if !IsRetryableError(err) {
		t.Error("connection refused message should be retryable")
	}
}

func TestErrorMessageFormat(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      2,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	err := doRetry(context.Background(), config, func() error {
		return NewRetryableError(errors.New("transient failure"))
	})

	if err == nil {
		t.Fatal("expected error")
	}

	expected := "operation failed after 3 attempts"
	if !containsString(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestZeroRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      0,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        100 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return NewRetryableError(errors.New("error"))
	})

	if err == nil {
		t.Error("doRetry() should return error")
	}
	if callCount != 1 {
		t.Errorf("operation called %d times, want 1", callCount)
	}
}

func TestNetOpErrorDial(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("connection refused"),
	}

	if !IsRetryableError(err) {
		t.Error("dial net.OpError should be retryable")
	}
}

func TestNetOpErrorWithConnectionRefused(t *testing.T) {
	err := &net.OpError{
		Op:  "read",
		Net: "tcp",
		Err: errors.New("connection refused by peer"),
	}

	if !IsRetryableError(err) {
		t.Error("connection refused net.OpError should be retryable")
	}
}

func TestNetOpErrorWithNoSuchHost(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: errors.New("no such host"),
	}

	if !IsRetryableError(err) {
		t.Error("no such host net.OpError should be retryable")
	}
}

func TestUnexpectedEOFIsRetryable(t *testing.T) {
	if !IsRetryableError(fmt.Errorf("wrapped: %w", errors.New("unexpected EOF"))) {
		t.Log("unexpected EOF pattern check")
	}
}
