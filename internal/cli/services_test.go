package cli

import (
	"testing"
)

func TestServicesListNoSystemd(t *testing.T) {
	// Listing services should not panic even if systemd isn't available.
	_, _, err := runCmd(t, servicesListCmd)
	if err != nil {
		// manager.ListServices returns nil error on failure in manager implementation,
		// but if Execute returns an error, ensure it's propagated.
		// Here just ensure command runs without panicking; allow non-nil err.
	}
}
