package node

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Amitesh-exp/NoInternetUpi/crypto"
	"github.com/Amitesh-exp/NoInternetUpi/models"
)

func generateTransactionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func CreateAndBroadcastPayment(senderUPI string, receiverUPI string, amount float64, publicKeyPath string, hasInternet bool, bankURL string) error {
	publicKey, err := crypto.LoadPublicKey(publicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	innerPayload := map[string]interface{}{
		"sender_upi":   senderUPI,
		"receiver_upi": receiverUPI,
		"amount":       amount,
	}

	innerBytes, err := json.Marshal(innerPayload)
	if err != nil {
		return fmt.Errorf("failed to serialize inner payload: %w", err)
	}

	encryptedPayload, encryptedAESKey, err := crypto.EncryptPayload(innerBytes, publicKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt payload: %w", err)
	}

	packet := models.PaymentPacket{
		TransactionID:    generateTransactionID(),
		SenderUPI:        senderUPI,
		ReceiverUPI:      receiverUPI,
		Amount:           amount,
		TTL:              time.Now().Add(5 * time.Minute),
		EncryptedPayload: encryptedPayload,
		EncryptedAESKey:  encryptedAESKey,
	}

	if hasInternet {
		fmt.Println("Node has internet — forwarding directly to bank")
		return ForwardToBank(packet, bankURL)
	}

	fmt.Println("Node has no internet — broadcasting to nearby nodes")
	return Broadcast(packet)
}