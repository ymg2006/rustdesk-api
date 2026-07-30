package jwt

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Jwt struct {
	Key                 []byte
	TokenExpireDuration time.Duration
}

type UserClaims struct {
	UserId uint `json:"user_id"`
	jwt.RegisteredClaims
}

func NewJwt(key string, tokenExpireDuration time.Duration) *Jwt {
	return &Jwt{
		Key:                 []byte(key),
		TokenExpireDuration: tokenExpireDuration,
	}
}

func (s *Jwt) GenerateToken(userId uint) string {
	if len(s.Key) == 0 {
		fmt.Println("jwt key is nil")
		return ""
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256,
		UserClaims{
			UserId: userId,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.TokenExpireDuration)),
			},
		})
	token, err := t.SignedString(s.Key)
	if err != nil {
		fmt.Printf("jwt token generate error: %v", err)
		return ""
	}
	return token
}

func (s *Jwt) ParseToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// SECURITY: Explicitly verify the signature algorithm to prevent algorithm confusion.
		// If it is not verified, the attacker can change alg to "none" or use the RSA public key to disguise it as an HMAC key to forge the token.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.Key, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims.UserId, nil
	}
	return 0, err
}

// MfaClaims is a temporary token (short-lived) used for two-step login verification.
type MfaClaims struct {
	UserId uint `json:"user_id"`
	jwt.RegisteredClaims
}

// GenerateMfaToken generates a temporary MFA token, valid for 5 minutes
func (s *Jwt) GenerateMfaToken(userId uint) string {
	if len(s.Key) == 0 {
		return ""
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, MfaClaims{
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
		},
	})
	token, err := t.SignedString(s.Key)
	if err != nil {
		return ""
	}
	return token
}

// ParseMfaToken parses MFA temporary token
func (s *Jwt) ParseMfaToken(tokenString string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenString, &MfaClaims{}, func(token *jwt.Token) (interface{}, error) {
		// SECURITY: Same as above, the MFA temporary token also needs to verify the signature algorithm to prevent forgery.
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.Key, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(*MfaClaims); ok && token.Valid {
		return claims.UserId, nil
	}
	return 0, err
}
