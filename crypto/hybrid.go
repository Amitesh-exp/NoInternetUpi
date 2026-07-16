package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"os"
)

// LoadPublicKey reads the bank's RSA public key from a .pem file
// Nodes use this to encrypt the AES key before broadcasting
func LoadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPub, nil
}

// LoadPrivateKey reads the bank's RSA private key from a .pem file
// Only the bank has this — used to decrypt the AES key
func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// EncryptPayload encrypts payment data using hybrid encryption
// Returns: encryptedPayload, encryptedAESKey, error
func EncryptPayload(payload []byte, bankPublicKey *rsa.PublicKey) ([]byte, []byte, error) {

	// Step 1: Generate a random 32 byte AES key
	// This key is unique per transaction — never reused
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, nil, err
	}

	// Step 2: Encrypt the payload using AES-GCM
	// GCM mode also detects tampering — if anyone modifies
	// even one byte of the encrypted payload, decryption fails
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	// Nonce is a random value that ensures same plaintext
	// encrypts to different ciphertext each time
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	// Seal appends the encrypted payload to the nonce
	// So encryptedPayload = nonce + ciphertext
	encryptedPayload := gcm.Seal(nonce, nonce, payload, nil)

	// Step 3: Encrypt the AES key using bank's RSA public key
	// Only the bank's private key can decrypt this
	encryptedAESKey, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		bankPublicKey,
		aesKey,
		nil,
	)
	if err != nil {
		return nil, nil, err
	}

	return encryptedPayload, encryptedAESKey, nil
}

// DecryptPayload decrypts the payment data — only the bank can do this
func DecryptPayload(encryptedPayload []byte, encryptedAESKey []byte, bankPrivateKey *rsa.PrivateKey) ([]byte, error) {

	// Step 1: Decrypt the AES key using bank's RSA private key
	aesKey, err := rsa.DecryptOAEP(
		sha256.New(),
		rand.Reader,
		bankPrivateKey,
		encryptedAESKey,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Step 2: Use the AES key to decrypt the payload
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// Split nonce and ciphertext
	nonceSize := gcm.NonceSize()
	if len(encryptedPayload) < nonceSize {
		return nil, errors.New("encrypted payload too short")
	}

	nonce, ciphertext := encryptedPayload[:nonceSize], encryptedPayload[nonceSize:]

	// Open decrypts and also verifies integrity
	// If anyone tampered with the payload, this returns an error
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed — payload may have been tampered with")
	}

	return plaintext, nil
}