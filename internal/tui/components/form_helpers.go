package components

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type RemoteLister interface {
	ListRootDirectories(remoteName string) ([]string, error)
}

func ValidateBufferSize(value string) error {
	if value == "" {
		return fmt.Errorf("buffer size cannot be empty")
	}

	matched, err := regexp.MatchString(`(?i)^\d+[kmg]$`, value)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid buffer size format: %q (expected format: number followed by K, M, or G, e.g., \"16M\", \"1G\", \"512K\")", value)
	}

	numStr := value[:len(value)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return fmt.Errorf("invalid number in buffer size: %q", value)
	}
	if num <= 0 {
		return fmt.Errorf("buffer size must be greater than 0: %q", value)
	}

	return nil
}

func ValidateDuration(value string) error {
	if value == "" {
		return fmt.Errorf("duration cannot be empty")
	}

	matched, err := regexp.MatchString(`(?i)^\d+[hms]$`, value)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid duration format: %q (expected format: number followed by h, m, or s, e.g., \"24h\", \"30m\", \"5s\")", value)
	}

	numStr := value[:len(value)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return fmt.Errorf("invalid number in duration: %q", value)
	}
	if num <= 0 {
		return fmt.Errorf("duration must be greater than 0: %q", value)
	}

	return nil
}

func ValidateUmask(value string) error {
	if value == "" {
		return fmt.Errorf("umask cannot be empty")
	}

	matched, err := regexp.MatchString(`^[0-7]{3,4}$`, value)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid umask format: %q (expected 3-4 digit octal number, e.g., \"002\", \"022\", \"0002\")", value)
	}

	return nil
}

func ValidateBandwidthLimit(value string) error {
	if value == "" {
		return nil
	}

	matched, err := regexp.MatchString(`(?i)^\d+[kmg]$`, value)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}
	if !matched {
		return fmt.Errorf("invalid bandwidth limit format: %q (expected format: number followed by K, M, or G, e.g., \"10M\", \"1G\", or leave empty for unlimited)", value)
	}

	numStr := value[:len(value)-1]
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return fmt.Errorf("invalid number in bandwidth limit: %q", value)
	}
	if num <= 0 {
		return fmt.Errorf("bandwidth limit must be greater than 0: %q", value)
	}

	return nil
}

func GetRemotePathSuggestions(rcloneClient interface{}, remoteName string, staticFallbacks []string) []string {
	var suggestions []string
	seen := make(map[string]bool)

	if rcloneClient != nil {
		if lister, ok := rcloneClient.(RemoteLister); ok {
			dirs, err := lister.ListRootDirectories(remoteName)
			if err == nil {
				for _, dir := range dirs {
					if dir == "" {
						continue
					}
					normalized := strings.TrimSuffix(dir, "/")
					if !seen[normalized] {
						seen[normalized] = true
						suggestions = append(suggestions, normalized)
					}
				}
			}
		}
	}

	for _, fallback := range staticFallbacks {
		if fallback == "" {
			continue
		}
		normalized := strings.TrimSuffix(fallback, "/")
		if !seen[normalized] {
			seen[normalized] = true
			suggestions = append(suggestions, fallback)
		}
	}

	return suggestions
}
