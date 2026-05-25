package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrTokenExpired     = errors.New("token is expired")
	ErrTokenInvalid     = errors.New("token is invalid")
	ErrTokenMalformed   = errors.New("token is malformed")
	ErrSignatureInvalid = errors.New("token signature is invalid")
)

// Claims расширяет стандартные JWT claims, если позже понадобятся кастомные поля
type Claims struct {
	jwt.RegisteredClaims
	Permissions []string `json:"permissions"`
}

type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
}

// NewJWTManager создаёт менеджер токенов
func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
	}
}

// Generate создаёт подписанный JWT для пользователя
// Алгоритм: HS256
func (m *JWTManager) Generate(userUUID uuid.UUID, permissions []string) (string, error) {
	now := time.Now()

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userUUID.String(),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.tokenDuration)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ParseWithClaims парсит токен и возвращает полные claims (для RBAC)
func (m *JWTManager) ParseWithClaims(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrTokenMalformed) {
			return nil, ErrTokenMalformed
		}
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, ErrSignatureInvalid
		}
		return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// Parse (старый метод) - оставляем для обратной совместимости
func (m *JWTManager) Parse(tokenString string) (uuid.UUID, error) {
	claims, err := m.ParseWithClaims(tokenString)
	if err != nil {
		return uuid.Nil, err
	}

	userUUID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid subject in token: %w", err)
	}

	return userUUID, nil
}

func (c *Claims) HasPermission(required string) bool {
	for _, perm := range c.Permissions {
		if perm == required {
			return true
		}
	}
	return false
}
