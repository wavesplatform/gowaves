package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"github.com/pkg/errors"
	"golang.org/x/crypto/argon2"
)

type crypt struct {
	key []byte
}

func NewCrypt(key []byte) *crypt {
	salt := []byte("E84265D411C08F99E092AE237F4EC250B2F20B2EAB7CFB2FCB0857880983DF44")
	pass := argon2.IDKey(key, salt, 4, 64*1024, 4, 32)
	return &crypt{
		key: pass,
	}
}

func (a *crypt) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) > 1024*1024 {
		return nil, errors.New("too big plaintext len for encrypting, 1MB limit exceeded")
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(ciphertext[aes.BlockSize:], plaintext)
	return ciphertext, nil
}

func (a *crypt) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}

	if byteLen := len(ciphertext); byteLen < aes.BlockSize {
		return nil, errors.Errorf("invalid cipher size %d", byteLen)
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	cipher.NewCFBDecrypter(block, iv).XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}
