package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// Lock / Unlock wallet request
type ManageWalletRequest struct {
	WalletPassword  string `json:"password"`
	UnlockTimestamp int    `json:"unlockTimestamp"`
}

func hashPassword(password string) ([]byte, error) {
	passwordBytes, err := hex.DecodeString(password)
	if err != nil {
		return nil, err
	}

	hkdfReader := hkdf.New(sha256.New, passwordBytes, []byte("Mnemonic Encryption Salt"), nil)
	aesKey := make([]byte, 32)
	_, err = io.ReadFull(hkdfReader, aesKey)
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
	return hex.EncodeToString(encryptedMnemonic), nil
}

func DecryptMnemonic(mnemonic []byte, password string) (string, error) {
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

	decrypted, err := gcm.Open(nil,
		mnemonic[:gcm.NonceSize()],
		mnemonic[gcm.NonceSize():],
		nil,
	)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(decrypted), nil
}