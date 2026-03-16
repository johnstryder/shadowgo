package recorder

import (
	"context"
	"testing"
)

func TestDetectRegionTools_ShouldReportAvailability(t *testing.T) {
	slurp, grim := DetectRegionTools()
	// At least one might be available on a dev machine; we just verify no panic
	_ = slurp
	_ = grim
}

func TestGetRegionFromSlurp_ShouldReturnErrorWhenSlurpUnavailable(t *testing.T) {
	// If slurp is available, this would block - skip in that case
	slurpAvail, _ := DetectRegionTools()
	if slurpAvail {
		t.Skip("slurp available - would block on user input")
	}

	_, err := GetRegionFromSlurp(context.Background())
	if err == nil {
		t.Error("expected error when slurp not available")
	}
}
