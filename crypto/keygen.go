package crypto

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

// GenerateAndSaveKeys generates RSA key pair and saves to disk
// Run this once to create bank_private.pem and bank_public.pem
func GenerateAndSaveKeys(privateKeyPath string, publicKeyPath string) error {

	// Generate 2048 bit RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	// Save private key
	privFile, err := os.Create(privateKeyPath)
	if err != nil {
		return err
	}
	defer privFile.Close()

	pem.Encode(privFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	// Save public key
	pubFile, err := os.Create(publicKeyPath)
	if err != nil {
		return err
	}
	defer pubFile.Close()

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	pem.Encode(pubFile, &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	return nil
}