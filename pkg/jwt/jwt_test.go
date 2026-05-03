package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testSecret = []byte("test-secret-key-32-bytes-long!!!")

func TestGenerateToken(t *testing.T) {
	userID := uuid.New()

	token, err := GenerateToken(userID, testSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestValidateToken(t *testing.T) {
	userID := uuid.New()

	token, err := GenerateToken(userID, testSecret)
	require.NoError(t, err)

	gotID, err := ValidateToken(token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, gotID)
}

func TestValidateToken_WrongSecret(t *testing.T) {
	userID := uuid.New()

	token, err := GenerateToken(userID, testSecret)
	require.NoError(t, err)

	_, err = ValidateToken(token, []byte("wrong-secret"))
	assert.Error(t, err)
}

func TestValidateToken_ExpiredToken(t *testing.T) {
	userID := uuid.New()

	// создаём токен с истёкшим сроком
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(testSecret)
	require.NoError(t, err)

	_, err = ValidateToken(signed, testSecret)
	assert.Error(t, err)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	_, err := ValidateToken("invalid.token.string", testSecret)
	assert.Error(t, err)
}

func TestValidateToken_WrongSigningMethod(t *testing.T) {
	userID := uuid.New()

	// подписываем RS256 вместо HS256
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"exp": time.Now().Add(TokenTTL).Unix(),
	}
	// генерируем с none алгоритмом
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = ValidateToken(signed, testSecret)
	assert.Error(t, err)
}
