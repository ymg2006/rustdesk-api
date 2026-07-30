package model

import (
	"testing"
	"time"
)

// TestSubscriptionStatus_None test never subscribed status
func TestSubscriptionStatus_None(t *testing.T) {
	u := &User{}
	if status := u.SubscriptionStatus(); status != "none" {
		t.Fatalf("expected 'none', got '%s'", status)
	}
	if u.IsSubscriptionActive() {
		t.Fatal("IsSubscriptionActive should be false for nil expire")
	}
	if days := u.SubscriptionDaysLeft(); days != 0 {
		t.Fatalf("expected 0 days left, got %d", days)
	}
}

// TestSubscriptionStatus_Active test subscription active status
func TestSubscriptionStatus_Active(t *testing.T) {
	future := time.Now().Add(30 * 24 * time.Hour)
	u := &User{
		SubscriptionPlan:     "pro",
		SubscriptionExpireAt: &future,
	}

	if status := u.SubscriptionStatus(); status != "active" {
		t.Fatalf("expected 'active', got '%s'", status)
	}
	if !u.IsSubscriptionActive() {
		t.Fatal("IsSubscriptionActive should be true")
	}
	if days := u.SubscriptionDaysLeft(); days <= 0 || days > 31 {
		t.Fatalf("expected days between 1-31, got %d", days)
	}
}

// TestSubscriptionStatus_Expired Test subscription expiration status
func TestSubscriptionStatus_Expired(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	u := &User{
		SubscriptionPlan:     "pro",
		SubscriptionExpireAt: &past,
	}

	if status := u.SubscriptionStatus(); status != "expired" {
		t.Fatalf("expected 'expired', got '%s'", status)
	}
	if u.IsSubscriptionActive() {
		t.Fatal("IsSubscriptionActive should be false for expired")
	}
	if days := u.SubscriptionDaysLeft(); days != 0 {
		t.Fatalf("expected 0 days left for expired, got %d", days)
	}
}

// TestSubscriptionDaysLeft test remaining days calculation
func TestSubscriptionDaysLeft(t *testing.T) {
	// Expires in exactly 7 days
	future := time.Now().Add(7 * 24 * time.Hour)
	u := &User{SubscriptionExpireAt: &future}
	days := u.SubscriptionDaysLeft()
	if days < 6 || days > 8 {
		t.Fatalf("expected ~7 days, got %d", days)
	}

	// Expires in exactly 1 day
	future2 := time.Now().Add(24 * time.Hour)
	u2 := &User{SubscriptionExpireAt: &future2}
	days2 := u2.SubscriptionDaysLeft()
	if days2 < 0 || days2 > 2 {
		t.Fatalf("expected ~1 day, got %d", days2)
	}
}

// TestSubscriptionStatus_ExactlyNow The test expires exactly now
func TestSubscriptionStatus_ExactlyNow(t *testing.T) {
	// Use microsecond precision
	justNow := time.Now()
	u := &User{SubscriptionExpireAt: &justNow}

	// Since Before is undefined on time.Now(), we just check and don't panic
	_ = u.SubscriptionStatus()
	_ = u.IsSubscriptionActive()
	_ = u.SubscriptionDaysLeft()
}
