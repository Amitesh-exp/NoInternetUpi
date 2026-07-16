package models

import "time"

type PaymentPacket struct {
	TransactionID    string    `json:"transaction_id"`
	SenderUPI        string    `json:"sender_upi"`
	ReceiverUPI      string    `json:"receiver_upi"`
	Amount           float64   `json:"amount"`
	TTL              time.Time `json:"ttl"`
	EncryptedPayload []byte    `json:"encrypted_payload"`
	EncryptedAESKey  []byte    `json:"encrypted_aes_key"`
}