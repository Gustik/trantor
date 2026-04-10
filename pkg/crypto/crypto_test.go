package crypto

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSalt(t *testing.T) {
	salt1, err := GenerateSalt()
	require.NoError(t, err)
	assert.Len(t, salt1, 32)

	salt2, err := GenerateSalt()
	require.NoError(t, err)

	// два salt не должны совпадать
	assert.NotEqual(t, salt1, salt2)
}

func TestGenerateMasterKey(t *testing.T) {
	key1, err := GenerateMasterKey()
	require.NoError(t, err)
	assert.Len(t, key1, KeySize)

	key2, err := GenerateMasterKey()
	require.NoError(t, err)

	// два ключа не должны совпадать
	assert.NotEqual(t, key1, key2)
}

func TestDeriveKeys(t *testing.T) {
	salt, err := GenerateSalt()
	require.NoError(t, err)

	authKey, encryptionKey := DeriveKeys("password123", salt)

	assert.Len(t, authKey, KeySize)
	assert.Len(t, encryptionKey, KeySize)

	// authKey и encryptionKey должны быть разными
	assert.NotEqual(t, authKey, encryptionKey)

	// одинаковый пароль + salt → одинаковые ключи (детерминированность)
	authKey2, encryptionKey2 := DeriveKeys("password123", salt)
	assert.Equal(t, authKey, authKey2)
	assert.Equal(t, encryptionKey, encryptionKey2)

	// разные пароли → разные ключи
	authKey3, _ := DeriveKeys("otherpassword", salt)
	assert.NotEqual(t, authKey, authKey3)
}

func TestEncryptDecrypt(t *testing.T) {
	key, err := GenerateMasterKey()
	require.NoError(t, err)

	plaintext := []byte("secret data")

	ciphertext, nonce, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, ciphertext)
	assert.NotEmpty(t, nonce)

	// зашифрованный текст не должен совпадать с исходным
	assert.NotEqual(t, plaintext, ciphertext)

	decrypted, err := Decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	_, _, err := Encrypt([]byte("short"), []byte("data"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestDecrypt_InvalidKeySize(t *testing.T) {
	_, err := Decrypt([]byte("short"), []byte("nonce"), []byte("ciphertext"))
	assert.ErrorIs(t, err, ErrInvalidKeySize)
}

func TestDecrypt_WrongKey(t *testing.T) {
	key, err := GenerateMasterKey()
	require.NoError(t, err)

	ciphertext, nonce, err := Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	wrongKey, err := GenerateMasterKey()
	require.NoError(t, err)

	_, err = Decrypt(wrongKey, nonce, ciphertext)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestDecrypt_CorruptedCiphertext(t *testing.T) {
	key, err := GenerateMasterKey()
	require.NoError(t, err)

	ciphertext, nonce, err := Encrypt(key, []byte("secret"))
	require.NoError(t, err)

	// портим зашифрованный текст
	ciphertext[0] ^= 0xff

	_, err = Decrypt(key, nonce, ciphertext)
	assert.ErrorIs(t, err, ErrDecryptionFailed)
}

func TestEncrypt_Nondeterministic(t *testing.T) {
	key, err := GenerateMasterKey()
	require.NoError(t, err)

	plaintext := []byte("secret")

	ciphertext1, nonce1, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	ciphertext2, nonce2, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	// каждое шифрование даёт разный nonce и ciphertext
	assert.NotEqual(t, nonce1, nonce2)
	assert.NotEqual(t, ciphertext1, ciphertext2)

	// но оба расшифровываются корректно
	decrypted1, err := Decrypt(key, nonce1, ciphertext1)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted1)

	decrypted2, err := Decrypt(key, nonce2, ciphertext2)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted2)
}

func TestDeriveKeys_DifferentSalts(t *testing.T) {
	salt1, err := GenerateSalt()
	require.NoError(t, err)

	salt2, err := GenerateSalt()
	require.NoError(t, err)

	authKey1, _ := DeriveKeys("password", salt1)
	authKey2, _ := DeriveKeys("password", salt2)

	// одинаковый пароль + разные salt → разные ключи
	assert.NotEqual(t, authKey1, authKey2)
}

// проверяем что ErrDecryptionFailed оборачивает оригинальную ошибку
func TestDecrypt_ErrorWrapping(t *testing.T) {
	key, err := GenerateMasterKey()
	require.NoError(t, err)

	_, err = Decrypt(key, make([]byte, 12), []byte("invalid ciphertext"))
	assert.ErrorIs(t, err, ErrDecryptionFailed)
	// оригинальная ошибка тоже доступна
	assert.False(t, errors.Is(err, ErrInvalidKeySize))
}
