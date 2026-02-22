// Package errors provides structured error types for the rclone-mount-sync application.
package errors

import (
	"errors"
	"fmt"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name: "basic error",
			err: &AppError{
				Code:       "TEST_001",
				Message:    "test error message",
				Suggestion: "try this",
			},
			expected: "test error message (code: TEST_001)",
		},
		{
			name: "error with cause",
			err: &AppError{
				Code:       "TEST_002",
				Message:    "wrapped error",
				Suggestion: "suggestion",
				Cause:      fmt.Errorf("underlying error"),
			},
			expected: "wrapped error (code: TEST_002): underlying error",
		},
		{
			name: "error without code",
			err: &AppError{
				Message:    "no code error",
				Suggestion: "suggestion",
			},
			expected: "no code error",
		},
		{
			name: "error with cause but no code",
			err: &AppError{
				Message:    "no code",
				Cause:      fmt.Errorf("cause"),
			},
			expected: "no code: cause",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &AppError{
		Code:    "TEST_001",
		Message: "test",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test with nil cause
	errNoCause := &AppError{
		Code:    "TEST_002",
		Message: "test",
	}
	if errNoCause.Unwrap() != nil {
		t.Errorf("Unwrap() with nil cause should return nil")
	}
}

func TestAppError_Is(t *testing.T) {
	// Test matching by code
	err1 := &AppError{Code: "RCLONE_001", Message: "first"}
	err2 := &AppError{Code: "RCLONE_001", Message: "second"}
	err3 := &AppError{Code: "RCLONE_002", Message: "first"}

	if !err1.Is(err2) {
		t.Error("errors with same code should match")
	}
	if err1.Is(err3) {
		t.Error("errors with different codes should not match")
	}

	// Test matching by message when no code
	err4 := &AppError{Message: "same message"}
	err5 := &AppError{Message: "same message"}
	err6 := &AppError{Message: "different message"}

	if !err4.Is(err5) {
		t.Error("errors with same message should match when no code")
	}
	if err4.Is(err6) {
		t.Error("errors with different messages should not match")
	}

	// Test with non-AppError target
	if err1.Is(fmt.Errorf("standard error")) {
		t.Error("should not match non-AppError")
	}
}

func TestAppError_FormatForTUI(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		contains []string
	}{
		{
			name: "full error",
			err: &AppError{
				Code:       "TEST_001",
				Message:    "test message",
				Suggestion: "try this solution",
			},
			contains: []string{"⚠ test message", "try this solution", "Error Code: TEST_001"},
		},
		{
			name: "error without suggestion",
			err: &AppError{
				Code:    "TEST_002",
				Message: "test message",
			},
			contains: []string{"⚠ test message", "Error Code: TEST_002"},
		},
		{
			name: "error without code",
			err: &AppError{
				Message:    "test message",
				Suggestion: "try this",
			},
			contains: []string{"⚠ test message", "try this"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.FormatForTUI()
			for _, s := range tt.contains {
				if !containsString(got, s) {
					t.Errorf("FormatForTUI() missing expected substring %q in:\n%s", s, got)
				}
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are properly defined
	sentinelErrors := []*AppError{
		ErrRcloneNotFound,
		ErrRcloneVersion,
		ErrNoRemotesConfigured,
		ErrMountPointExists,
		ErrMountPointNotFound,
		ErrServiceNotFound,
		ErrServiceFailed,
		ErrConfigInvalid,
		ErrPermissionDenied,
		ErrRcloneError,
	}

	for _, err := range sentinelErrors {
		if err.Code == "" {
			t.Errorf("sentinel error %v has empty code", err)
		}
		if err.Message == "" {
			t.Errorf("sentinel error %v has empty message", err)
		}
		if err.Suggestion == "" {
			t.Errorf("sentinel error %v has empty suggestion", err)
		}
	}
}

func TestNewRcloneNotFoundError(t *testing.T) {
	cause := fmt.Errorf("not in PATH")
	err := NewRcloneNotFoundError(cause)

	if err.Code != ErrRcloneNotFound.Code {
		t.Errorf("expected code %s, got %s", ErrRcloneNotFound.Code, err.Code)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
	if err.Message != ErrRcloneNotFound.Message {
		t.Errorf("expected message %s, got %s", ErrRcloneNotFound.Message, err.Message)
	}
}

func TestNewRcloneVersionError(t *testing.T) {
	cause := fmt.Errorf("version check failed")
	err := NewRcloneVersionError("1.50.0", "1.60.0", cause)

	if err.Code != ErrRcloneVersion.Code {
		t.Errorf("expected code %s, got %s", ErrRcloneVersion.Code, err.Code)
	}
	if !containsString(err.Message, "1.50.0") || !containsString(err.Message, "1.60.0") {
		t.Errorf("expected message to contain version info, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewNoRemotesConfiguredError(t *testing.T) {
	cause := fmt.Errorf("config empty")
	err := NewNoRemotesConfiguredError(cause)

	if err.Code != ErrNoRemotesConfigured.Code {
		t.Errorf("expected code %s, got %s", ErrNoRemotesConfigured.Code, err.Code)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewMountPointExistsError(t *testing.T) {
	cause := fmt.Errorf("mount busy")
	err := NewMountPointExistsError("/mnt/test", cause)

	if err.Code != ErrMountPointExists.Code {
		t.Errorf("expected code %s, got %s", ErrMountPointExists.Code, err.Code)
	}
	if !containsString(err.Message, "/mnt/test") {
		t.Errorf("expected message to contain mount point, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewMountPointNotFoundError(t *testing.T) {
	cause := fmt.Errorf("stat failed")
	err := NewMountPointNotFoundError("/mnt/missing", cause)

	if err.Code != ErrMountPointNotFound.Code {
		t.Errorf("expected code %s, got %s", ErrMountPointNotFound.Code, err.Code)
	}
	if !containsString(err.Message, "/mnt/missing") {
		t.Errorf("expected message to contain mount point, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewServiceNotFoundError(t *testing.T) {
	cause := fmt.Errorf("unit not loaded")
	err := NewServiceNotFoundError("rclone-mount-test", cause)

	if err.Code != ErrServiceNotFound.Code {
		t.Errorf("expected code %s, got %s", ErrServiceNotFound.Code, err.Code)
	}
	if !containsString(err.Message, "rclone-mount-test") {
		t.Errorf("expected message to contain service name, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewServiceFailedError(t *testing.T) {
	cause := fmt.Errorf("exit code 1")
	err := NewServiceFailedError("start", "rclone-mount-test", cause)

	if err.Code != ErrServiceFailed.Code {
		t.Errorf("expected code %s, got %s", ErrServiceFailed.Code, err.Code)
	}
	if !containsString(err.Message, "start") || !containsString(err.Message, "rclone-mount-test") {
		t.Errorf("expected message to contain operation and service name, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewConfigInvalidError(t *testing.T) {
	cause := fmt.Errorf("yaml parse error")
	err := NewConfigInvalidError("missing required field 'name'", cause)

	if err.Code != ErrConfigInvalid.Code {
		t.Errorf("expected code %s, got %s", ErrConfigInvalid.Code, err.Code)
	}
	if !containsString(err.Message, "missing required field") {
		t.Errorf("expected message to contain details, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewPermissionDeniedError(t *testing.T) {
	cause := fmt.Errorf("EACCES")
	err := NewPermissionDeniedError("write", "/etc/config", cause)

	if err.Code != ErrPermissionDenied.Code {
		t.Errorf("expected code %s, got %s", ErrPermissionDenied.Code, err.Code)
	}
	if !containsString(err.Message, "write") || !containsString(err.Message, "/etc/config") {
		t.Errorf("expected message to contain operation and resource, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestNewRcloneError(t *testing.T) {
	cause := fmt.Errorf("exit status 1")
	err := NewRcloneError("rclone mount gdrive: /mnt/gdrive", cause)

	if err.Code != ErrRcloneError.Code {
		t.Errorf("expected code %s, got %s", ErrRcloneError.Code, err.Code)
	}
	if !containsString(err.Message, "rclone mount") {
		t.Errorf("expected message to contain command, got %s", err.Message)
	}
	if err.Cause != cause {
		t.Errorf("expected cause %v, got %v", cause, err.Cause)
	}
}

func TestIsAppError(t *testing.T) {
	appErr := &AppError{Code: "TEST", Message: "test"}
	stdErr := fmt.Errorf("standard error")

	if !IsAppError(appErr) {
		t.Error("IsAppError should return true for AppError")
	}
	if IsAppError(stdErr) {
		t.Error("IsAppError should return false for standard error")
	}
}

func TestGetAppError(t *testing.T) {
	appErr := &AppError{Code: "TEST", Message: "test"}
	stdErr := fmt.Errorf("standard error")

	result := GetAppError(appErr)
	if result != appErr {
		t.Error("GetAppError should return the same AppError")
	}

	result = GetAppError(stdErr)
	if result != nil {
		t.Error("GetAppError should return nil for non-AppError")
	}

	result = GetAppError(nil)
	if result != nil {
		t.Error("GetAppError should return nil for nil error")
	}
}

func TestWrap(t *testing.T) {
	t.Run("wrap AppError", func(t *testing.T) {
		inner := &AppError{
			Code:       "INNER_001",
			Message:    "inner message",
			Suggestion: "inner suggestion",
			Cause:      fmt.Errorf("root cause"),
		}
		wrapped := Wrap(inner, "outer context")

		if wrapped.Code != "INNER_001" {
			t.Errorf("expected code INNER_001, got %s", wrapped.Code)
		}
		if !containsString(wrapped.Message, "outer context") || !containsString(wrapped.Message, "inner message") {
			t.Errorf("expected wrapped message, got %s", wrapped.Message)
		}
		if wrapped.Cause != inner.Cause {
			t.Error("cause should be preserved from inner error")
		}
	})

	t.Run("wrap standard error", func(t *testing.T) {
		stdErr := fmt.Errorf("standard error")
		wrapped := Wrap(stdErr, "context")

		if wrapped.Code != "GEN_001" {
			t.Errorf("expected code GEN_001, got %s", wrapped.Code)
		}
		if !containsString(wrapped.Message, "context") {
			t.Errorf("expected message to contain context, got %s", wrapped.Message)
		}
		if wrapped.Cause != stdErr {
			t.Error("cause should be the original error")
		}
	})

	t.Run("wrap nil", func(t *testing.T) {
		wrapped := Wrap(nil, "context")
		if wrapped != nil {
			t.Error("wrapping nil should return nil")
		}
	})
}

func TestFormatErrorForTUI(t *testing.T) {
	t.Run("AppError", func(t *testing.T) {
		appErr := &AppError{
			Code:       "TEST_001",
			Message:    "test message",
			Suggestion: "try this",
		}
		got := FormatErrorForTUI(appErr)
		if !containsString(got, "⚠ test message") {
			t.Errorf("expected formatted output, got %s", got)
		}
	})

	t.Run("standard error", func(t *testing.T) {
		stdErr := fmt.Errorf("standard error")
		got := FormatErrorForTUI(stdErr)
		if !containsString(got, "⚠ standard error") {
			t.Errorf("expected formatted output, got %s", got)
		}
		if !containsString(got, "unexpected error") {
			t.Errorf("expected generic suggestion, got %s", got)
		}
	})

	t.Run("nil error", func(t *testing.T) {
		got := FormatErrorForTUI(nil)
		if got != "" {
			t.Errorf("expected empty string for nil, got %s", got)
		}
	})
}

func TestErrorsIs(t *testing.T) {
	// Test that errors.Is works with our AppError type
	cause := fmt.Errorf("underlying cause")
	err := NewRcloneNotFoundError(cause)

	// Should match sentinel by code
	if !errors.Is(err, ErrRcloneNotFound) {
		t.Error("errors.Is should match sentinel error by code")
	}

	// Should not match different error
	if errors.Is(err, ErrRcloneVersion) {
		t.Error("errors.Is should not match different error")
	}
}

func TestErrorsAs(t *testing.T) {
	// Test that errors.As works with our AppError type
	err := NewMountPointExistsError("/mnt/test", nil)

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Error("errors.As should extract AppError")
	}

	if appErr.Code != ErrMountPointExists.Code {
		t.Errorf("expected code %s, got %s", ErrMountPointExists.Code, appErr.Code)
	}
}

func TestErrorsUnwrap(t *testing.T) {
	// Test that errors.Unwrap works with our AppError type
	cause := fmt.Errorf("underlying cause")
	err := NewRcloneNotFoundError(cause)

	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("errors.Unwrap should return the cause")
	}
}

// Helper function
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
