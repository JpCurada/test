package utils

import (
	"fmt"
	"strconv"
	"time"

	"github.com/ISKOnnect/iskonnect-web/internal/models"
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID    int  `json:"user_id"`
	IsStudent bool `json:"is_student"`
	jwt.RegisteredClaims
}

func GenerateJWT(user *models.User, secret string, expiryHours int) (string, error) {
	claims := &JWTClaims{
		UserID:    user.ID,
		IsStudent: user.IsStudent,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiryHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   strconv.Itoa(user.ID),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateJWT(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
