package service

import (
	"testing"
	"time"

	"github.com/pquerna/otp/totp"
	"github.com/sirupsen/logrus"
	"github.com/ymg2006/rustdesk-api/v2/config"
	"github.com/ymg2006/rustdesk-api/v2/model"
)

// TestVerifyMfaCode_Regression regression test: covering the correct/error code and empty key guard of MFA dynamic code verification,
// and clock skew (Skew hardening) scenarios. No database required, pure memory verification.
func TestVerifyMfaCode_Regression(t *testing.T) {
	// Fixed Logger.Warnf to avoid nil panic in tests
	Logger = logrus.New()

	us := &UserService{}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "RustDesk", AccountName: "alice"})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}

	u := &model.User{Username: "alice", MfaEnabled: true, MfaSecret: key.Secret()}

	// 1) Correct dynamic code should pass
	code, err := totp.GenerateCode(u.MfaSecret, time.Now())
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if !us.VerifyMfaCode(u, code) {
		t.Fatalf("Correct dynamic code%qshould be verified and passed", code)
	}

	// 2) Wrong dynamic codes should be rejected
	if us.VerifyMfaCode(u, "000000") {
		t.Fatalf("Error dynamic code should not pass")
	}

	// 3) Empty keys should be rejected by the guard (and issue an alarm), and there will be no silent misjudgment
	empty := &model.User{MfaEnabled: true, MfaSecret: ""}
	if us.VerifyMfaCode(empty, code) {
		t.Fatalf("Empty keys should not pass validation")
	}

	// 4) In the scenario where the clock offset is 60s (±30s beyond the default Skew=1), the hardened Skew=3 (±90s) should be able to pass
	skewedCode, err := totp.GenerateCode(u.MfaSecret, time.Now().Add(-60*time.Second))
	if err != nil {
		t.Fatalf("GenerateCode(skewed): %v", err)
	}
	// The old behavior (standard totp.Validate, Skew=1) must fail at this offset - reproducing the original bug scenario
	if totp.Validate(skewedCode, u.MfaSecret) {
		t.Fatalf("The old logic was expected to fail at 60s offset, but actually passed (inconsistent with the root cause)")
	}
	// The new behavior (VerifyMfaCode, Skew=3) should pass under this offset - verify hardening takes effect
	if !us.VerifyMfaCode(u, skewedCode) {
		t.Fatalf("After reinforcement, VerifyMfaCode should be able to pass under 60s offset.")
	}
}

// TestVerifyMfaCode_ConfigurableSkew verifies that mfa_totp_skew configuration takes effect:
// Operation and maintenance can adjust the TOTP clock tolerance through mfa_totp_skew in config.yaml (Skew=N → tolerance ±30s×N),
// No need to recompile. This test covers both the "enlarged tolerance to accept larger excursions" and the "default tolerance to reject excessive excursions" situations.
func TestVerifyMfaCode_ConfigurableSkew(t *testing.T) {
	// Avoid Logger.Warnf nil panic in tests
	Logger = logrus.New()
	defer func() { Config = nil }() // Restore the overall situation to avoid contaminating other tests

	us := &UserService{}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "RustDesk", AccountName: "alice"})
	if err != nil {
		t.Fatalf("totp.Generate: %v", err)
	}
	u := &model.User{Username: "alice", MfaEnabled: true, MfaSecret: key.Secret()}

	const offset = 150 * time.Second // 150s offset: Only Skew>=5 (±150s) can be tolerated

	// 1) Configure a larger tolerance Skew=6 (±180s): the 150s offset should pass
	Config = &config.Config{MfaTotpSkew: 6}
	code, err := totp.GenerateCode(u.MfaSecret, time.Now().Add(-offset))
	if err != nil {
		t.Fatalf("GenerateCode(skewed): %v", err)
	}
	if !us.VerifyMfaCode(u, code) {
		t.Fatalf("When configuring Skew=6, the verification code with%voffset should pass", offset)
	}

	// 2) Control: fall back to default Skew=3 (±90s) < 150s: should be rejected
	Config = &config.Config{MfaTotpSkew: 3}
	if us.VerifyMfaCode(u, code) {
		t.Fatalf("When the default Skew=3, the verification code with%voffset should not pass", offset)
	}

	// 3) Boundary: Configuring illegal values ​​(0 / negative numbers) should fall back to the default Skew=3, and the 150s offset is still rejected
	Config = &config.Config{MfaTotpSkew: 0}
	if us.VerifyMfaCode(u, code) {
		t.Fatalf("Illegal configuration Skew=0 should fall back to default (3), and the verification code with%voffset should not pass", offset)
	}
}
