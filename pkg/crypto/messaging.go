package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"

	"github.com/pkg/errors"
	"golang.org/x/crypto/curve25519"
)

const (
	messageProtocolVersion byte = 1
	messageHeaderSize           = 1 + KeySize + aes.BlockSize + 2*KeySize + aes.BlockSize
)

func SharedKey(sk SecretKey, pk PublicKey, prefix []byte) ([]byte, error) {
	k, err := curve25519.X25519(sk[:], pk[:])
	if err != nil {
		return nil, err
	}
	h1 := sha256.New()
	if _, err := h1.Write(prefix); err != nil {
		return nil, err
	}
	hashedPrefix := h1.Sum(nil)
	h := hmac.New(sha256.New, hashedPrefix)
	if _, err := h.Write(k[:]); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func Encrypt(key, message []byte) ([]byte, error) {
	buf := make([]byte, messageHeaderSize+len(message))
	buf[0] = messageProtocolVersion
	sessionKey := make([]byte, KeySize)
	_, err := rand.Read(sessionKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, err
	}
	encryptedMessage, err := encryptAESCTR(message, sessionKey, iv)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt message")
	}
	encryptedSessionKey, err := encryptAESECB(sessionKey, key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encrypt session key")
	}
	h := hmac.New(sha256.New, sessionKey)
	if _, err := h.Write(message); err != nil {
		return nil, err
	}
	messageHMAC := h.Sum(nil)
	h = hmac.New(sha256.New, key)
	if _, err := h.Write(sessionKey); err != nil {
		return nil, err
	}
	if _, err := h.Write(iv); err != nil {
		return nil, err
	}
	sessionKeyHMAC := h.Sum(nil)
	copy(buf[1:], encryptedSessionKey)
	copy(buf[1+KeySize+aes.BlockSize:], sessionKeyHMAC)
	copy(buf[1+KeySize+aes.BlockSize+KeySize:], messageHMAC)
	copy(buf[1+KeySize+aes.BlockSize+KeySize+KeySize:], encryptedMessage)
	return buf, nil
}

func Decrypt(key, encrypted []byte) ([]byte, error) {
	if encrypted[0] != messageProtocolVersion {
		return nil, errors.Errorf("invalid message protocol version, must be %d", messageProtocolVersion)
	}
	if len(encrypted) < messageHeaderSize {
		return nil, errors.Errorf("invalid message length")
	}
	sessionKey, err := decryptAESECB(encrypted[1:1+KeySize+aes.BlockSize], key)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decrypt session key")
	}
	iv := encrypted[1+KeySize+aes.BlockSize+2*KeySize : 1+KeySize+aes.BlockSize+2*KeySize+aes.BlockSize]
	enc := encrypted[1+KeySize+aes.BlockSize+2*KeySize:]
	h := hmac.New(sha256.New, key)
	if _, err := h.Write(sessionKey); err != nil {
		return nil, errors.Wrap(err, "h.Write() failed")
	}
	if _, err := h.Write(iv); err != nil {
		return nil, errors.Wrap(err, "h.Write() failed")
	}
	expectedSessionKeyHMAC := h.Sum(nil)
	if !hmac.Equal(expectedSessionKeyHMAC, encrypted[1+KeySize+aes.BlockSize:1+KeySize+aes.BlockSize+KeySize]) {
		return nil, errors.New("invalid message authentication code")
	}
	message, err := decryptAESCTR(enc, sessionKey, iv)
	if err != nil {
		return nil, errors.New("failed to decrypt message")
	}
	h = hmac.New(sha256.New, sessionKey)
	if _, err := h.Write(message); err != nil {
		return nil, errors.Wrap(err, "h.Write() failed")
	}
	expectedMessageHMAC := h.Sum(nil)
	if !hmac.Equal(expectedMessageHMAC, encrypted[1+KeySize+aes.BlockSize+KeySize:1+KeySize+aes.BlockSize+KeySize+KeySize]) {
		return nil, errors.New("invalid message authentication code")
	}
	return message, nil
}

func encryptAESECB(message, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	message, err = padPKCS7Padding(message)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, len(message))
	enc := buf[:]
	for len(message) > 0 {
		block.Encrypt(enc, message)
		message = message[aes.BlockSize:]
		enc = enc[aes.BlockSize:]
	}
	return buf, nil
}

func decryptAESECB(encrypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, len(encrypted))
	msg := buf[:]
	for len(encrypted) > 0 {
		block.Decrypt(msg, encrypted)
		encrypted = encrypted[aes.BlockSize:]
		msg = msg[aes.BlockSize:]
	}
	msg, err = trimPKCS7Padding(buf)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func encryptAESCTR(message, key, iv []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize {
		return nil, errors.New("invalid IV length, must be the same as AES block size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	encrypted := make([]byte, aes.BlockSize+len(message))
	copy(encrypted, iv)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(encrypted[aes.BlockSize:], message)
	return encrypted, nil
}

func decryptAESCTR(encrypted, key, iv []byte) ([]byte, error) {
	if len(iv) != aes.BlockSize {
		return nil, errors.New("invalid IV length, must be the same as AES block size")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(block, iv)
	message := make([]byte, len(encrypted)-aes.BlockSize)
	stream.XORKeyStream(message, encrypted[aes.BlockSize:])
	return message, nil
}

func padPKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("unable to pad empty data")
	}
	n := aes.BlockSize - (len(data) % aes.BlockSize)
	buf := make([]byte, len(data)+n)
	copy(buf, data)
	copy(buf[len(data):], bytes.Repeat([]byte{byte(n)}, n))
	return buf, nil
}

func trimPKCS7Padding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("unable to trim empty data")
	}
	if len(data)%aes.BlockSize != 0 {
		return nil, errors.New("invalid PKCS7 padding: data length is not multiple of AES block size")
	}
	c := data[len(data)-1]
	n := int(c)
	if n == 0 || n > len(data) {
		return nil, errors.Errorf("invalid PKCS7 padding: invalid padding byte value %d", n)
	}
	for i := 0; i < n; i++ {
		if b := data[len(data)-n+i]; b != c {
			return nil, errors.Errorf("invalid PKCS7 padding: unexpected padding byte value %d", b)
		}
	}
	return data[:len(data)-n], nil
}
