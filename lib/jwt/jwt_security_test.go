package jwt

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// hmacSign signs the signingString with HMAC-SHA256 and returns the RawURLEncoding result (simulating an attacker's obfuscated signature using the "public key" as the key).
func hmacSign(signingString string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(signingString))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// buildUnsignedNoneToken Manually constructs a JWT of alg:none (in the form of header.payload., leaving the signature blank).
// This is the most naive form of forged token: the attacker can impersonate any user_id without providing any signature.
func buildUnsignedNoneToken(t *testing.T, userID uint) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"user_id":%d,"exp":%d}`, userID, time.Now().Add(time.Hour).Unix()),
	))
	return header + "." + payload + "."
}

// TestParseToken_HS256_HappyPath confirms that normal HS256 tokens can be parsed correctly (fix should not break normal functionality).
func TestParseToken_HS256_HappyPath(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)
	token := svc.GenerateToken(12345)
	if token == "" {
		t.Fatal("HS256 Token generation failed")
	}
	uid, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("Normal HS256 token parsing should succeed, actual error:%v", err)
	}
	if uid != 12345 {
		t.Fatalf("Expected to resolve user_id=12345, actual%d", uid)
	}
}

// TestParseToken_NoneAlgorithm_Manual Manually constructed alg:none tokens must be rejected (must not resolve to a valid user_id).
func TestParseToken_NoneAlgorithm_Manual(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)
	noneTok := buildUnsignedNoneToken(t, 999)
	uid, err := svc.ParseToken(noneTok)
	if err == nil {
		t.Fatalf("alg:none token should not be accepted, but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected alg:none token should return uid 0, actual%d", uid)
	}
}

// TestParseToken_NoneAlgorithm_Library uses the jwt library to construct a token with SigningMethodNone, which must also be rejected.
func TestParseToken_NoneAlgorithm_Library(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)
	claims := UserClaims{
		UserId: 7,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("Failed to construct alg:none token:%v", err)
	}
	uid, err := svc.ParseToken(signed)
	if err == nil {
		t.Fatalf("alg:none token should not be accepted, but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected alg:none token should return uid 0, actual%d", uid)
	}
}

// TestParseToken_RS256Rejected RS256 tokens signed with an RSA private key must be rejected (algorithm obfuscation protection).
func TestParseToken_RS256Rejected(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pk))
	if err != nil {
		t.Fatalf("Failed to parse RSA private key for testing:%v", err)
	}
	claims := UserClaims{
		UserId: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	rs256Token, err := tok.SignedString(rsaKey)
	if err != nil {
		t.Fatalf("Failed to construct RS256 token:%v", err)
	}

	uid, err := svc.ParseToken(rs256Token)
	if err == nil {
		t.Fatalf("RS256 token should not be accepted (algorithm mismatch), but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected RS256 token should return uid 0, actual%d", uid)
	}
}

// TestParseToken_RS256ConfusionWithHmacSecret simulates classic algorithm confusion attack:
// The attacker sets alg to RS256 and uses HMAC to sign with the "public key". Even if the signature can be verified by HMAC,
// Must also be rejected due to algorithm mismatch (verify keyfunc method before returning the key).
func TestParseToken_RS256ConfusionWithHmacSecret(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)

	claims := UserClaims{
		UserId: 42,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	// Forged into RS256 header, but HMAC signed with server "public/key" string:
	// Hand-assembled "RS256 header + HMAC signature" obfuscated token (bypassing the RS256 limitation of requiring an RSA private key).
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(
		fmt.Sprintf(`{"user_id":%d,"exp":%d}`, claims.UserId, claims.ExpiresAt.Unix()),
	))
	signingString := header + "." + payload
	// Use HMAC-SHA256 (using test-secret as the key) to create a signature string that "can pass HMAC verification"
	mac := hmacSign(signingString, []byte("test-secret"))
	confusedToken := signingString + "." + mac

	uid, err := svc.ParseToken(confusedToken)
	if err == nil {
		t.Fatalf("Algorithm obfuscation attack (RS256 header + HMAC signature) should not be accepted, but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected obfuscated token should return uid 0, actual%d", uid)
	}
}

// TestParseMfaToken_NoneAlgorithm MFA temporary tokens must also reject alg:none.
func TestParseMfaToken_NoneAlgorithm(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)
	noneTok := buildUnsignedNoneToken(t, 555)
	uid, err := svc.ParseMfaToken(noneTok)
	if err == nil {
		t.Fatalf("MFA alg:none token should not be accepted, but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected MFA alg:none token should return uid 0, actual%d", uid)
	}
}

// TestParseMfaToken_RS256Rejected MFA temporary tokens must also reject RS256.
func TestParseMfaToken_RS256Rejected(t *testing.T) {
	svc := NewJwt("test-secret", time.Hour)

	rsaKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(pk))
	if err != nil {
		t.Fatalf("Failed to parse RSA private key for testing:%v", err)
	}
	claims := MfaClaims{
		UserId: 88,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	rs256Token, err := tok.SignedString(rsaKey)
	if err != nil {
		t.Fatalf("Failed to construct MFA RS256 token:%v", err)
	}
	uid, err := svc.ParseMfaToken(rs256Token)
	if err == nil {
		t.Fatalf("MFA RS256 token should not be accepted, but parsed successfully (uid=%d)", uid)
	}
	if uid != 0 {
		t.Fatalf("Rejected MFA RS256 token should return uid 0, actual%d", uid)
	}
}

// TestParseMfaToken_HS256_HappyPath confirms that normal MFA HS256 tokens can be parsed correctly.
func TestParseMfaToken_HS256_HappyPath(t *testing.T) {
	svc := NewJwt("test-secret", 5*time.Minute)
	token := svc.GenerateMfaToken(777)
	if token == "" {
		t.Fatal("MFA HS256 token generation failed")
	}
	uid, err := svc.ParseMfaToken(token)
	if err != nil {
		t.Fatalf("Normal MFA HS256 token parsing should succeed, actual error:%v", err)
	}
	if uid != 777 {
		t.Fatalf("Expected to resolve user_id=777, actual%d", uid)
	}
}
