package rclone

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"time"
)

const (
	DefaultMaxRetries      = 3
	DefaultInitialDelay    = 500 * time.Millisecond
	DefaultMaxDelay        = 30 * time.Second
	DefaultRetryMultiplier = 2.0
)

type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	RetryMultiplier float64
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      DefaultMaxRetries,
		InitialDelay:    DefaultInitialDelay,
		MaxDelay:        DefaultMaxDelay,
		RetryMultiplier: DefaultRetryMultiplier,
	}
}

type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

func NewRetryableError(err error) error {
	return &RetryableError{Err: err}
}

type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string {
	return e.Err.Error()
}

func (e *PermanentError) Unwrap() error {
	return e.Err
}

func NewPermanentError(err error) error {
	return &PermanentError{Err: err}
}

func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var retryableErr *RetryableError
	if errors.As(err, &retryableErr) {
		return true
	}

	var permanentErr *PermanentError
	if errors.As(err, &permanentErr) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" {
			return true
		}
		if strings.Contains(opErr.Error(), "connection refused") {
			return true
		}
		if strings.Contains(opErr.Error(), "no such host") {
			return true
		}
		if strings.Contains(opErr.Error(), "temporary failure") {
			return true
		}
	}

	errStr := strings.ToLower(err.Error())
	retryablePatterns := []string{
		"timeout",
		"timed out",
		"connection refused",
		"connection reset",
		"connection closed",
		"no such host",
		"dns",
		"temporary failure",
		"network is unreachable",
		"host is unreachable",
		"i/o timeout",
		"deadline exceeded",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	return false
}

func IsPermanentError(err error) bool {
	if err == nil {
		return false
	}

	var permanentErr *PermanentError
	return errors.As(err, &permanentErr)
}

func classifyExitError(err error) error {
	if err == nil {
		return nil
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return err
	}

	stderr := strings.ToLower(string(exitErr.Stderr))
	exitCode := exitErr.ExitCode()

	permanentPatterns := []string{
		"config file not found",
		"configuration not found",
		"no config",
		"authentication failed",
		"access denied",
		"permission denied",
		"invalid config",
		"unknown remote",
		"remote not found",
		"invalid credentials",
		"unauthorized",
		"forbidden",
		"not found",
	}

	for _, pattern := range permanentPatterns {
		if strings.Contains(stderr, pattern) {
			return NewPermanentError(err)
		}
	}

	if exitCode == 1 && strings.Contains(stderr, "error") {
		return err
	}

	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network",
		"dns",
		"temporary",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(stderr, pattern) {
			return NewRetryableError(err)
		}
	}

	return err
}

type Operation func() error

func doRetry(ctx context.Context, config RetryConfig, op Operation) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := op()
		if err == nil {
			return nil
		}

		err = classifyExitError(err)
		lastErr = err

		if !IsRetryableError(err) {
			return err
		}

		if attempt == config.MaxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * config.RetryMultiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	if lastErr != nil {
		return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
	}
	return errors.New("operation failed")
}

type bytesOperation func() ([]byte, error)

func doRetryBytes(ctx context.Context, config RetryConfig, op bytesOperation) ([]byte, error) {
	var lastErr error
	var result []byte
	delay := config.InitialDelay

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		res, err := op()
		if err == nil {
			return res, nil
		}

		err = classifyExitError(err)
		lastErr = err

		if !IsRetryableError(err) {
			return result, err
		}

		if attempt == config.MaxRetries {
			break
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * config.RetryMultiplier)
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
	}

	if lastErr != nil {
		return result, fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
	}
	return result, errors.New("operation failed")
}
