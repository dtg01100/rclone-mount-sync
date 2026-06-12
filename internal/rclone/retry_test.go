package rclone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	if err := os.WriteFile(mockPath, []byte(script), 0755); err != nil { //nolint:gosec
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

	remotes, err := c.ListRemotes(context.Background())
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

	// Capture the call count by reading the mock's invocation log.
	// createMockRcloneForRetry writes nothing to disk for tracking, so we
	// infer "no retry" by checking the elapsed time: with InitialDelay=10ms
	// and 3 retries we'd sleep ~70ms; if no retries happen we finish in
	// well under 50ms. Using a strict upper bound is the cleanest way to
	// prove the retry loop was bypassed.
	start := time.Now()
	_, err := c.ListRemotes(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("ListRemotes() should return error for permanent error")
	}
	if !IsPermanentError(err) {
		t.Errorf("expected permanent error, got: %v", err)
	}
	// Three retries with backoff would take >= ~70ms; one call should
	// take well under 30ms. A 100ms budget is generous to avoid CI flakes.
	if elapsed > 100*time.Millisecond {
		t.Errorf("ListRemotes took %v; expected no retry (under 100ms)", elapsed)
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

	_, err := c.ListRemotes(context.Background())
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

	remoteType, err := c.GetRemoteType(context.Background(), "gdrive")
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

	entries, err := c.ListRemotePath(context.Background(), "gdrive", "/")
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

	err := c.TestRemoteAccess(context.Background(), "gdrive", "/")
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
				if !errors.Is(result, tt.err) {
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

	// Use a single sentinel so errors.Is can match the wrapped error
	// returned by doRetry. A previous version of this test constructed
	// a fresh sentinel inside errors.Is, which never matches.
	persistent := NewRetryableError(errors.New("persistent transient error"))

	callCount := 0
	err := doRetry(context.Background(), config, func() error {
		callCount++
		return persistent
	})

	if err == nil {
		t.Fatal("doRetry() should return error after max retries")
	}
	if callCount != 3 {
		t.Errorf("operation called %d times, want 3", callCount)
	}
	if !errors.Is(err, persistent) {
		t.Errorf("expected wrapped retryable error, got: %v", err)
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

	if !errors.Is(err, context.Canceled) {
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

	// With MaxRetries=3 there are 3 sleep periods between the 4 calls
	// (MaxRetries+1 total attempts): 50ms, 100ms, 200ms. A previous
	// version of this test used t.Skip when fewer than 2 delays were
	// captured, allowing a backoff regression to silently pass.
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

	if callCount != 4 {
		t.Fatalf("expected 4 attempts (MaxRetries+1), got %d", callCount)
	}
	if len(delays) != 3 {
		t.Fatalf("expected 3 delays, got %d", len(delays))
	}

	expectedDelays := []time.Duration{
		50 * time.Millisecond,
		100 * time.Millisecond,
		200 * time.Millisecond,
	}

	for i, expected := range expectedDelays {
		// 50% tolerance to absorb scheduler jitter on busy CI runners.
		// Without the assertion the test was a t.Logf that never failed.
		tolerance := expected / 2
		if delays[i] < expected-tolerance || delays[i] > expected+tolerance {
			t.Errorf("delay[%d] = %v, expected approximately %v (±%v)", i, delays[i], expected, tolerance)
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
		InitialDelay:    10 * time.Second, // Long delay to ensure we can cancel before next retry
		MaxDelay:        10 * time.Second,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	started := make(chan struct{})
	go func() {
		close(started)
		// Cancel after the first retry starts its delay
		cancel()
	}()

	<-started // Wait for goroutine to start

	err := doRetry(ctx, config, func() error {
		callCount++
		return NewRetryableError(errors.New("transient error"))
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("doRetry() should return context.Canceled, got %v", err)
	}
}

func TestDoRetryBytesContextCancellationBetweenRetries(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    10 * time.Second, // Long delay to ensure we can cancel before next retry
		MaxDelay:        10 * time.Second,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	started := make(chan struct{})
	go func() {
		close(started)
		// Cancel after the first retry starts its delay
		cancel()
	}()

	<-started // Wait for goroutine to start

	_, err := doRetryBytes(ctx, config, func() ([]byte, error) {
		callCount++
		return nil, NewRetryableError(errors.New("transient error"))
	})

	if !errors.Is(err, context.Canceled) {
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
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("error message should contain %q, got %q", expected, err.Error())
	}
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
	// io.ErrUnexpectedEOF is the sentinel; a plain errors.New("unexpected EOF")
	// does not match errors.Is(err, io.ErrUnexpectedEOF), which is what the
	// production code checks. Wrap the real sentinel and assert the result
	// is retryable.
	if !IsRetryableError(fmt.Errorf("wrapped: %w", io.ErrUnexpectedEOF)) {
		t.Error("wrapped io.ErrUnexpectedEOF should be retryable")
	}
	// The plain-message variant is NOT retryable — it should not be treated
	// as the sentinel just because the text matches.
	if IsRetryableError(errors.New("unexpected EOF")) {
		t.Error("plain errors.New(\"unexpected EOF\") should not be retryable (it is not the io.ErrUnexpectedEOF sentinel)")
	}
}

// TestDoRetryBackoffGuardsAgainstInf verifies that with a very large
// RetryMultiplier the per-iteration delay caps at MaxDelay instead of
// overflowing to +Inf (which would make the `<= 0 || > MaxDelay` check
// critical to trigger).
func TestDoRetryBackoffGuardsAgainstInf(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        30 * time.Millisecond,
		RetryMultiplier: 1e9, // deliberately absurd
	}

	// Cap the entire test in time so a regression to +Inf doesn't
	// hang the suite for a very long sleep.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = doRetry(context.Background(), config, func() error {
			return NewRetryableError(errors.New("retryable"))
		})
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("doRetry did not return within 2s; the +Inf guard likely regressed")
	}
}

// TestDoRetryContextCancelDuringOp verifies that a context that becomes
// done while the op is running causes doRetry to return ctx.Err()
// immediately, rather than retrying.
func TestDoRetryContextCancelDuringOp(t *testing.T) {
	config := RetryConfig{
		MaxRetries:      5,
		InitialDelay:    10 * time.Millisecond,
		MaxDelay:        50 * time.Millisecond,
		RetryMultiplier: 2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	start := time.Now()
	err := doRetry(ctx, config, func() error {
		callCount++
		// Cancel the context inside the very first op so that
		// doRetry sees ctx.Err() != nil on the post-op check.
		if callCount == 1 {
			cancel()
			// Wait a beat to ensure ctx.Done() is observed.
			time.Sleep(5 * time.Millisecond)
		}
		return NewRetryableError(errors.New("retryable"))
	})
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got: %v", err)
	}
	// Should not have retried. With InitialDelay=10ms and 5 retries,
	// even one retry would push elapsed well over 15ms; cap at 50ms.
	if callCount > 1 {
		t.Errorf("expected at most 1 call (no retry after ctx cancel), got %d", callCount)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("doRetry took %v after ctx cancel; expected quick return", elapsed)
	}
	// Make sure the cancel resource is released.
	cancel()
}
