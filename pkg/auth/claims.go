package auth

import (
	"github.com/golang-jwt/jwt/v5"
)

type accessTokenClaims struct {
	jwt.RegisteredClaims

	UID int64 `json:"uid"`
}
