package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

type crypt struct {
	key []byte
}

func NewCrypt(key []byte) *crypt {
	return &crypt{
		key: sha5000(key),
	}
}

func (a *crypt) Encrypt(plaintext []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("invalid cipher size %d", byteLen)
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	cipher.NewCFBDecrypter(block, iv).XORKeyStream(ciphertext, ciphertext)

	return ciphertext, nil
}

func _sha256(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

func sha5000(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	for i := 0; i < 5000; i++ {
		out = _sha256(out)
	}
	return out
}
