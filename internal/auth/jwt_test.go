package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestManager(t *testing.T) *JWTManager {
	t.Helper()
	return NewJWTManager("test-secret-key-for-jwt-unit-tests", time.Hour)
}

func TestJWTManager_GenerateAndParse_Success(t *testing.T) {
	mgr := setupTestManager(t)
	originalUUID := uuid.New()

	token, err := mgr.Generate(originalUUID)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	parsedUUID, err := mgr.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, originalUUID, parsedUUID)
}

func TestJWTManager_GenerateAndParseWithClaims_Success(t *testing.T) {
	mgr := setupTestManager(t)
	originalUUID := uuid.New()

	token, err := mgr.Generate(originalUUID)
	require.NoError(t, err)

	claims, err := mgr.ParseWithClaims(token)
	require.NoError(t, err)
	assert.NotNil(t, claims)

	parsedUUID, err := uuid.Parse(claims.Subject)
	require.NoError(t, err)
	assert.Equal(t, originalUUID, parsedUUID)

	assert.Empty(t, claims.Permissions)
}

func TestClaims_HasPermission_Success(t *testing.T) {
	claims := &Claims{
		Permissions: []string{"manage_tasks", "view_project"},
	}

	assert.True(t, claims.HasPermission("manage_tasks"))
	assert.True(t, claims.HasPermission("view_project"))
	assert.False(t, claims.HasPermission("manage_roles"))
}

func TestClaims_HasPermission_EmptyPermissions(t *testing.T) {
	claims := &Claims{
		Permissions: []string{},
	}

	assert.False(t, claims.HasPermission("manage_tasks"))
}

func TestClaims_HasPermission_NilPermissions(t *testing.T) {
	claims := &Claims{
		Permissions: nil,
	}

	assert.False(t, claims.HasPermission("manage_tasks"))
}

func TestJWTManager_Parse_Expired(t *testing.T) {
	mgr := NewJWTManager("test-secret", time.Millisecond*10)
	originalUUID := uuid.New()

	token, err := mgr.Generate(originalUUID)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 20)

	_, err = mgr.Parse(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWTManager_ParseWithClaims_Expired(t *testing.T) {
	mgr := NewJWTManager("test-secret", time.Millisecond*10)
	originalUUID := uuid.New()

	token, err := mgr.Generate(originalUUID)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 20)

	_, err = mgr.ParseWithClaims(token)
	assert.ErrorIs(t, err, ErrTokenExpired)
}

func TestJWTManager_Parse_InvalidSignature(t *testing.T) {
	mgr1 := NewJWTManager("secret-1", time.Hour)
	mgr2 := NewJWTManager("secret-2", time.Hour)
	originalUUID := uuid.New()

	token, err := mgr1.Generate(originalUUID)
	require.NoError(t, err)

	_, err = mgr2.Parse(token)
	assert.ErrorIs(t, err, ErrSignatureInvalid)
}

func TestJWTManager_Parse_Malformed(t *testing.T) {
	mgr := setupTestManager(t)

	_, err := mgr.Parse("not-a-jwt-string")
	assert.ErrorIs(t, err, ErrTokenMalformed)
}

func TestJWTManager_Parse_InvalidSubject(t *testing.T) {
	mgr := setupTestManager(t)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "not-a-uuid-string",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		Permissions: []string{"view_project"},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString([]byte("test-secret-key-for-jwt-unit-tests"))

	_, err := mgr.Parse(tokenStr)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid subject in token")
}

func TestJWTManager_TokenContainsEmptyPermissions(t *testing.T) {
	mgr := setupTestManager(t)
	userUUID := uuid.New()

	token, err := mgr.Generate(userUUID)
	require.NoError(t, err)

	parsedToken, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret-key-for-jwt-unit-tests"), nil
	})
	require.NoError(t, err)

	claims, ok := parsedToken.Claims.(*Claims)
	require.True(t, ok)

	assert.Empty(t, claims.Permissions)
	assert.Equal(t, userUUID.String(), claims.Subject)
}

func TestJWTManager_EmptyPermissions(t *testing.T) {
	mgr := setupTestManager(t)
	userUUID := uuid.New()

	token, err := mgr.Generate(userUUID)
	require.NoError(t, err)

	claims, err := mgr.ParseWithClaims(token)
	require.NoError(t, err)

	assert.Empty(t, claims.Permissions)
	assert.Equal(t, userUUID.String(), claims.Subject)
}
