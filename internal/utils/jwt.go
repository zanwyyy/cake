package utils

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	accessSecret  = []byte("your-access-secret")  
	refreshSecret = []byte("your-refresh-secret") 

	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(accessSecret)
}

func GenerateRefreshToken(userID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(refreshTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(refreshSecret)
}

func ValidateAccessToken(tokenStr string) (*Claims, error) {

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid access token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func ValidateRefreshToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid refresh token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func RefreshAccessToken(refreshToken string) (string, error) {
	claims, err := ValidateRefreshToken(refreshToken)
	if err != nil {
		return "", err
	}
	return GenerateAccessToken(claims.UserID)
}

func Logout(tokenStr string) error {
	claims, err := ValidateAccessToken(tokenStr)
	if err != nil {
		return err
	}
	if claims.ExpiresAt == nil {
		return errors.New("token has no expiry")
	}
	return nil
}

func ExtractUserID(tokenStr string) (string, error) {
	claims, err := ValidateAccessToken(tokenStr)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
