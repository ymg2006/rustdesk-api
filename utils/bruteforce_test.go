package utils

import (
	"testing"
	"time"
)

// TestBruteForce_BanThresholdReached verifies ban threshold logic based on in-memory per-IP counters:
// The IP is banned after continuous failures reach the threshold, and the ban status can be read through CheckSecurityStatus.
// This mechanism is the core of "brute force cracking of IP bans" (http/middleware/limiter.gocalls this LoginLimiter).
func TestBruteForce_BanThresholdReached(t *testing.T) {
	threshold := 3
	ll := NewLoginLimiter(SecurityPolicy{
		BanThreshold: threshold,
		BanDuration:  5 * time.Minute,
	})
	ip := "203.0.113.5"

	// Should not be banned until threshold is reached
	for i := 0; i < threshold-1; i++ {
		ll.RecordFailedAttempt(ip)
		if banned, _ := ll.CheckSecurityStatus(ip); banned {
			t.Fatalf("IP should not be banned before reaching threshold (attempt %d)", i+1)
		}
	}

	// Threshold reached -> should be banned
	ll.RecordFailedAttempt(ip)
	if banned, _ := ll.CheckSecurityStatus(ip); !banned {
		t.Fatalf("IP should be banned after %d failed attempts", threshold)
	}

	// Different IPs do not affect each other (per-IP memory map)
	if banned, _ := ll.CheckSecurityStatus("198.51.100.7"); banned {
		t.Fatalf("different IP should not be affected by another IP's ban")
	}

	// Banning is disabled when BanThreshold <= 0: will never be banned
	disabled := NewLoginLimiter(SecurityPolicy{BanThreshold: 0, CaptchaThreshold: -1})
	for i := 0; i < 10; i++ {
		disabled.RecordFailedAttempt(ip)
	}
	if banned, _ := disabled.CheckSecurityStatus(ip); banned {
		t.Fatalf("with BanThreshold=0 and CaptchaThreshold<0, IP must not be banned (default off)")
	}
}
