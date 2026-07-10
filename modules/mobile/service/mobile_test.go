package service_test

import (
	"testing"
)

// generateOTP is tested via exported wrapper below.
func TestGenerateOTP_Length(t *testing.T) {
	for _, length := range []int{4, 6, 8} {
		code, err := ExportGenerateOTP(length)
		if err != nil {
			t.Fatalf("generateOTP(%d): %v", length, err)
		}
		if len(code) != length {
			t.Errorf("expected length %d, got %d (code=%q)", length, len(code), code)
		}
	}
}

func TestGenerateOTP_NumericOnly(t *testing.T) {
	code, _ := ExportGenerateOTP(6)
	for i, ch := range code {
		if ch < '0' || ch > '9' {
			t.Errorf("position %d: expected digit, got %q", i, ch)
		}
	}
}

func TestGenerateOTP_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, _ := ExportGenerateOTP(6)
		seen[code] = true
	}
	// With 10^6 possible values, 100 samples should not all be the same
	if len(seen) == 1 {
		t.Error("OTP generator produced 100 identical values — likely broken RNG")
	}
}
