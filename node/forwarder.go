package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Amitesh-exp/NoInternetUpi/models"
)

type PaymentRequest struct {
	Packet models.PaymentPacket `json:"packet"`
}

type PaymentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func ForwardToBank(packet models.PaymentPacket, bankURL string) error {
	req := PaymentRequest{Packet: packet}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	resp, err := http.Post(bankURL+"/pay", "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to reach bank: %w", err)
	}
	defer resp.Body.Close()

	var paymentResp PaymentResponse
	err = json.NewDecoder(resp.Body).Decode(&paymentResp)
	if err != nil {
		return fmt.Errorf("failed to decode bank response: %w", err)
	}

	if !paymentResp.Success {
		return fmt.Errorf("bank rejected payment: %s", paymentResp.Message)
	}

	fmt.Println("Payment forwarded successfully:", paymentResp.Message)
	return nil
}