package service_test

import (
	"testing"

	svc "github.com/breadgarlicbigint/bread-golang-boilerplate/modules/appversion/service"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		// Patch
		{"2.4.1", "2.4.2", -1},
		{"2.4.2", "2.4.1", 1},
		{"2.4.1", "2.4.1", 0},
		// Minor
		{"2.3.9", "2.4.0", -1},
		{"2.4.0", "2.3.9", 1},
		// Major
		{"1.9.9", "2.0.0", -1},
		{"2.0.0", "1.9.9", 1},
		// v prefix
		{"v2.0.0", "2.0.0", 0},
		// Zeros
		{"0.0.1", "0.0.2", -1},
		{"1.0.0", "1.0.0", 0},
		// Pre-release stripped
		{"2.4.1-beta", "2.4.1", 0},
	}

	for _, tt := range tests {
		got, err := svc.ExportCompareVersions(tt.a, tt.b) // exported wrapper for test
		if err != nil {
			t.Errorf("compareVersions(%q, %q): unexpected error: %v", tt.a, tt.b, err)
			continue
		}
		if got != tt.want {
			t.Errorf("compareVersions(%q, %q) = %d; want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestVersionCheckStatus(t *testing.T) {
	tests := []struct {
		name          string
		clientVersion string
		minVersion    string
		currentVer    string
		forceUpdate   bool
		wantStatus    svc.UpdateStatus
	}{
		{
			name: "up to date",
			clientVersion: "2.5.0", minVersion: "2.0.0", currentVer: "2.5.0",
			forceUpdate: true, wantStatus: svc.UpToDate,
		},
		{
			name: "soft update available",
			clientVersion: "2.3.0", minVersion: "2.0.0", currentVer: "2.5.0",
			forceUpdate: false, wantStatus: svc.UpdateAvailable,
		},
		{
			name: "forced update required",
			clientVersion: "1.9.0", minVersion: "2.0.0", currentVer: "2.5.0",
			forceUpdate: true, wantStatus: svc.UpdateRequired,
		},
		{
			name: "below min but force disabled → available",
			clientVersion: "1.9.0", minVersion: "2.0.0", currentVer: "2.5.0",
			forceUpdate: false, wantStatus: svc.UpdateAvailable,
		},
		{
			name: "exactly at min version",
			clientVersion: "2.0.0", minVersion: "2.0.0", currentVer: "2.5.0",
			forceUpdate: true, wantStatus: svc.UpdateAvailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := svc.ExportDetermineStatus(tt.clientVersion, tt.minVersion, tt.currentVer, tt.forceUpdate)
			if status != tt.wantStatus {
				t.Errorf("status = %q; want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestParseSemVerInvalid(t *testing.T) {
	_, err := svc.ExportCompareVersions("not-a-version", "1.0.0")
	if err == nil {
		t.Error("expected error for invalid version string")
	}
}
