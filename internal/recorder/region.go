package recorder

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Region holds screen region coordinates from slurp.
type Region struct {
	X, Y, W, H int
}

// DetectRegionTools checks if slurp and grim are available for region selection.
func DetectRegionTools() (slurpAvailable, grimAvailable bool) {
	if _, err := exec.LookPath("slurp"); err == nil {
		slurpAvailable = true
	}
	if _, err := exec.LookPath("grim"); err == nil {
		grimAvailable = true
	}
	return slurpAvailable, grimAvailable
}

// GetRegionFromSlurp runs slurp to let the user select a screen region.
// Returns nil if slurp is not available or user cancels.
func GetRegionFromSlurp(ctx context.Context) (*Region, error) {
	slurpAvailable, _ := DetectRegionTools()
	if !slurpAvailable {
		return nil, fmt.Errorf("slurp not found in PATH")
	}

	cmd := exec.CommandContext(ctx, "slurp", "-f", "%x,%y,%w,%h")
	out, err := cmd.Output()
	if err != nil {
		// User cancellation typically results in non-zero exit
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("unexpected slurp output: %s", string(out))
	}

	var r Region
	if r.X, err = strconv.Atoi(parts[0]); err != nil {
		return nil, err
	}
	if r.Y, err = strconv.Atoi(parts[1]); err != nil {
		return nil, err
	}
	if r.W, err = strconv.Atoi(parts[2]); err != nil {
		return nil, err
	}
	if r.H, err = strconv.Atoi(parts[3]); err != nil {
		return nil, err
	}

	return &r, nil
}
