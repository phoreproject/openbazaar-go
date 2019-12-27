package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Lock / Unlock wallet request
type ManageWalletRequest struct {
	WalletPassword  string `json:"password"`
	UnlockTimestamp int    `json:"unlockTimestamp,omitempty"`
}

func hashPassword(password string) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, []byte(password), []byte("Mnemonic Encryption Salt"), nil)
	aesKey := make([]byte, 32)
	_, err := io.ReadFull(hkdfReader, aesKey)
	if err != nil {
		return nil, err
	}
	return aesKey, nil
}

func EncryptMnemonic(mnemonic string, password string) (string, error) {
	aesKey, err := hashPassword(password)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return "", err
	}

	encryptedMnemonic := gcm.Seal(nonce, nonce, []byte(mnemonic), nil)
	return string(encryptedMnemonic), nil
}

func DecryptMnemonic(mnemonic string, password string) (string, error) {
	aesKey, err := hashPassword(password)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	if len(mnemonic) < gcm.NonceSize() {
		return "", errors.New("malformed ciphertext")
	}

	mnemonicBytes := []byte(mnemonic)

	decrypted, err := gcm.Open(nil,
		mnemonicBytes[:gcm.NonceSize()],
		mnemonicBytes[gcm.NonceSize():],
		nil,
	)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}