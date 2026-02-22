// Package errors provides structured error types for the rclone-mount-sync application.
// These errors include codes, messages, and user-friendly suggestions to improve
// the user experience when errors occur.
package errors

import (
	"fmt"
	"strings"
)

// AppError represents a structured application error with additional context.
// It implements the error interface and supports error wrapping and comparison.
type AppError struct {
	// Code is a unique identifier for the error type (e.g., "RCLONE_001")
	Code string

	// Message is a brief description of the error
	Message string

	// Suggestion provides actionable guidance for the user
	Suggestion string

	// Cause is the underlying error that caused this error (optional)
	Cause error
}

// Error implements the error interface and returns a formatted error message.
func (e *AppError) Error() string {
	var sb strings.Builder

	sb.WriteString(e.Message)

	if e.Code != "" {
		sb.WriteString(" (code: ")
		sb.WriteString(e.Code)
		sb.WriteString(")")
	}

	if e.Cause != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Cause.Error())
	}

	return sb.String()
}

// Unwrap returns the underlying cause of the error, enabling error unwrapping.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// Is checks if the target error matches this error type.
// This enables errors.Is() comparisons for AppError types.
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	// Match by code if both have codes
	if e.Code != "" && t.Code != "" {
		return e.Code == t.Code
	}
	// Otherwise match by message
	return e.Message == t.Message
}

// FormatForTUI returns a formatted string suitable for display in the TUI.
// The output includes the error message, suggestion, and code in a user-friendly format.
func (e *AppError) FormatForTUI() string {
	var sb strings.Builder

	sb.WriteString("⚠ ")
	sb.WriteString(e.Message)
	sb.WriteString("\n\n")

	if e.Suggestion != "" {
		sb.WriteString(e.Suggestion)
		sb.WriteString("\n\n")
	}

	if e.Code != "" {
		sb.WriteString("Error Code: ")
		sb.WriteString(e.Code)
	}

	return sb.String()
}

// --- Sentinel Errors ---

var (
	// ErrRcloneNotFound indicates that the rclone binary was not found.
	ErrRcloneNotFound = &AppError{
		Code:       "RCLONE_001",
		Message:    "rclone is not installed",
		Suggestion: "Install rclone using your package manager or from https://rclone.org/install/",
	}

	// ErrRcloneVersion indicates that the installed rclone version is too old.
	ErrRcloneVersion = &AppError{
		Code:       "RCLONE_003",
		Message:    "rclone version is too old",
		Suggestion: "Upgrade rclone to version 1.60.0 or later. Visit https://rclone.org/install/ for instructions.",
	}

	// ErrNoRemotesConfigured indicates that no rclone remotes are configured.
	ErrNoRemotesConfigured = &AppError{
		Code:       "RCLONE_002",
		Message:    "No rclone remotes are configured",
		Suggestion: "Run 'rclone config' to set up a remote first, or use 'rclone config file' to locate your configuration.",
	}

	// ErrMountPointExists indicates that the mount point is already in use.
	ErrMountPointExists = &AppError{
		Code:       "VAL_001",
		Message:    "Mount point is already in use",
		Suggestion: "Choose a different mount point or unmount the existing mount first using 'fusermount -u <mount-point>'",
	}

	// ErrMountPointNotFound indicates that the mount point does not exist.
	ErrMountPointNotFound = &AppError{
		Code:       "VAL_003",
		Message:    "Mount point does not exist",
		Suggestion: "Create the mount point directory first, or check that the path is correct.",
	}

	// ErrServiceNotFound indicates that a systemd service was not found.
	ErrServiceNotFound = &AppError{
		Code:       "SYS_002",
		Message:    "Systemd service not found",
		Suggestion: "The service may not have been created yet. Check the service name or create the service first.",
	}

	// ErrServiceFailed indicates that a service operation failed.
	ErrServiceFailed = &AppError{
		Code:       "SYS_003",
		Message:    "Service operation failed",
		Suggestion: "Check the service logs using 'journalctl --user -u <service-name>' for more details.",
	}

	// ErrConfigInvalid indicates a configuration validation error.
	ErrConfigInvalid = &AppError{
		Code:       "CFG_001",
		Message:    "Configuration is invalid",
		Suggestion: "Check your configuration file for errors. Use the settings screen to reconfigure.",
	}

	// ErrPermissionDenied indicates a permission denied error.
	ErrPermissionDenied = &AppError{
		Code:       "PERM_001",
		Message:    "Permission denied",
		Suggestion: "Ensure you have the necessary permissions for this operation. You may need to adjust file permissions or run with elevated privileges.",
	}

	// ErrRcloneError indicates that an rclone command failed.
	ErrRcloneError = &AppError{
		Code:       "RCLONE_004",
		Message:    "rclone command failed",
		Suggestion: "Check the rclone logs for details. Verify your remote configuration and network connectivity.",
	}
)

// --- Constructor Functions ---

// NewRcloneNotFoundError creates a new ErrRcloneNotFound error with an optional cause.
func NewRcloneNotFoundError(cause error) *AppError {
	return &AppError{
		Code:       ErrRcloneNotFound.Code,
		Message:    ErrRcloneNotFound.Message,
		Suggestion: ErrRcloneNotFound.Suggestion,
		Cause:      cause,
	}
}

// NewRcloneVersionError creates a new ErrRcloneVersion error with version details.
func NewRcloneVersionError(currentVersion string, minVersion string, cause error) *AppError {
	return &AppError{
		Code:       ErrRcloneVersion.Code,
		Message:    fmt.Sprintf("rclone version %s is too old (minimum: %s)", currentVersion, minVersion),
		Suggestion: ErrRcloneVersion.Suggestion,
		Cause:      cause,
	}
}

// NewNoRemotesConfiguredError creates a new ErrNoRemotesConfigured error with an optional cause.
func NewNoRemotesConfiguredError(cause error) *AppError {
	return &AppError{
		Code:       ErrNoRemotesConfigured.Code,
		Message:    ErrNoRemotesConfigured.Message,
		Suggestion: ErrNoRemotesConfigured.Suggestion,
		Cause:      cause,
	}
}

// NewMountPointExistsError creates a new ErrMountPointExists error with mount point details.
func NewMountPointExistsError(mountPoint string, cause error) *AppError {
	return &AppError{
		Code:       ErrMountPointExists.Code,
		Message:    fmt.Sprintf("Mount point %q is already in use", mountPoint),
		Suggestion: ErrMountPointExists.Suggestion,
		Cause:      cause,
	}
}

// NewMountPointNotFoundError creates a new ErrMountPointNotFound error with mount point details.
func NewMountPointNotFoundError(mountPoint string, cause error) *AppError {
	return &AppError{
		Code:       ErrMountPointNotFound.Code,
		Message:    fmt.Sprintf("Mount point %q does not exist", mountPoint),
		Suggestion: ErrMountPointNotFound.Suggestion,
		Cause:      cause,
	}
}

// NewServiceNotFoundError creates a new ErrServiceNotFound error with service details.
func NewServiceNotFoundError(serviceName string, cause error) *AppError {
	return &AppError{
		Code:       ErrServiceNotFound.Code,
		Message:    fmt.Sprintf("Service %q not found", serviceName),
		Suggestion: ErrServiceNotFound.Suggestion,
		Cause:      cause,
	}
}

// NewServiceFailedError creates a new ErrServiceFailed error with operation details.
func NewServiceFailedError(operation string, serviceName string, cause error) *AppError {
	return &AppError{
		Code:       ErrServiceFailed.Code,
		Message:    fmt.Sprintf("Failed to %s service %q", operation, serviceName),
		Suggestion: ErrServiceFailed.Suggestion,
		Cause:      cause,
	}
}

// NewConfigInvalidError creates a new ErrConfigInvalid error with validation details.
func NewConfigInvalidError(details string, cause error) *AppError {
	return &AppError{
		Code:       ErrConfigInvalid.Code,
		Message:    fmt.Sprintf("Configuration is invalid: %s", details),
		Suggestion: ErrConfigInvalid.Suggestion,
		Cause:      cause,
	}
}

// NewPermissionDeniedError creates a new ErrPermissionDenied error with operation details.
func NewPermissionDeniedError(operation string, resource string, cause error) *AppError {
	return &AppError{
		Code:       ErrPermissionDenied.Code,
		Message:    fmt.Sprintf("Permission denied for %s on %s", operation, resource),
		Suggestion: ErrPermissionDenied.Suggestion,
		Cause:      cause,
	}
}

// NewRcloneError creates a new ErrRcloneError with command details.
func NewRcloneError(command string, cause error) *AppError {
	return &AppError{
		Code:       ErrRcloneError.Code,
		Message:    fmt.Sprintf("rclone command failed: %s", command),
		Suggestion: ErrRcloneError.Suggestion,
		Cause:      cause,
	}
}

// --- Helper Functions ---

// IsAppError checks if an error is an AppError type.
func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

// GetAppError attempts to extract an AppError from an error.
// Returns the AppError if found, or nil otherwise.
func GetAppError(err error) *AppError {
	if err == nil {
		return nil
	}
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return nil
}

// Wrap wraps an existing error with additional context.
// If the error is already an AppError, it returns a new AppError with the same code
// but with the additional message context.
func Wrap(err error, message string) *AppError {
	if err == nil {
		return nil
	}

	if appErr, ok := err.(*AppError); ok {
		return &AppError{
			Code:       appErr.Code,
			Message:    message + ": " + appErr.Message,
			Suggestion: appErr.Suggestion,
			Cause:      appErr.Cause,
		}
	}

	return &AppError{
		Code:       "GEN_001",
		Message:    message,
		Suggestion: "Check the error details and try again.",
		Cause:      err,
	}
}

// FormatErrorForTUI formats any error for display in the TUI.
// If the error is an AppError, it uses FormatForTUI(). Otherwise, it provides
// a generic formatted output.
func FormatErrorForTUI(err error) string {
	if err == nil {
		return ""
	}

	if appErr, ok := err.(*AppError); ok {
		return appErr.FormatForTUI()
	}

	// Generic error formatting
	return fmt.Sprintf("⚠ %s\n\nAn unexpected error occurred. Check the logs for more details.", err.Error())
}
