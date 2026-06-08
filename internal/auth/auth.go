// Package auth реализует JWT-аутентификацию пользователей.
package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ErrInvalidToken возвращается при невалидном или просроченном JWT.
var ErrInvalidToken = errors.New("invalid token")

// Claims описывает JWT-claims пользователя GophKeeper.
type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Login  string    `json:"login"`
	jwt.RegisteredClaims
}

// Manager управляет выпуском и проверкой JWT-токенов.
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// NewManager создаёт менеджер JWT с указанным секретом и временем жизни токена.
func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{
		secret: []byte(secret),
		ttl:    ttl,
	}
}

// GenerateToken выпускает JWT для авторизованного пользователя.
func (m *Manager) GenerateToken(userID uuid.UUID, login string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Login:  login,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			IssuedAt:  jwt.NewNumericDate(now),
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

// ParseToken проверяет JWT и возвращает claims пользователя.
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}
