package main
import (
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Amitesh-exp/NoInternetUpi/bank"
	"github.com/Amitesh-exp/NoInternetUpi/crypto"
	"github.com/Amitesh-exp/NoInternetUpi/models"
	_ "modernc.org/sqlite"
)

type BankServer struct {
	accounts    *bank.AccountStore
	idempotency *bank.IdempotencyStore
	privateKey  *rsa.PrivateKey
}

type PaymentRequest struct {
	Packet models.PaymentPacket `json:"packet"`
}

type PaymentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	db, err := sql.Open("sqlite", "bank.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	accountStore, err := bank.NewAccountStore(db)
	if err != nil {
		log.Fatal("Failed to create account store:", err)
	}

	err = accountStore.Seed()
	if err != nil {
		log.Fatal("Failed to seed accounts:", err)
	}

	idempotencyStore, err := bank.NewIdempotencyStore(db)
	if err != nil {
		log.Fatal("Failed to create idempotency store:", err)
	}

	privateKey, err := crypto.LoadPrivateKey("bank_private.pem")
	if err != nil {
		log.Fatal("Failed to load private key:", err)
	}

	server := &BankServer{
		accounts:    accountStore,
		idempotency: idempotencyStore,
		privateKey:  privateKey,
	}

	http.HandleFunc("/pay", server.handlePayment)
	http.HandleFunc("/balance", server.handleBalance)

	fmt.Println("Bank server running on port 8080")
	fmt.Println("Test accounts: alice@upi (1000), bob@upi (500), charlie@upi (750)")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (s *BankServer) handlePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PaymentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respond(w, false, "invalid request format", http.StatusBadRequest)
		return
	}

	packet := req.Packet

	if time.Now().After(packet.TTL) {
		respond(w, false, "packet expired — TTL exceeded", http.StatusBadRequest)
		return
	}

	alreadyProcessed, err := s.idempotency.HasBeenProcessed(packet.TransactionID)
	if err != nil {
		respond(w, false, "internal error", http.StatusInternalServerError)
		return
	}
	if alreadyProcessed {
		respond(w, true, "already processed", http.StatusOK)
		return
	}

	decryptedBytes, err := crypto.DecryptPayload(packet.EncryptedPayload, packet.EncryptedAESKey, s.privateKey)
	if err != nil {
		respond(w, false, "decryption failed — packet may be tampered", http.StatusBadRequest)
		return
	}

	var innerPayload map[string]interface{}
	err = json.Unmarshal(decryptedBytes, &innerPayload)
	if err != nil {
		respond(w, false, "invalid inner payload", http.StatusBadRequest)
		return
	}

	senderUPI, ok1 := innerPayload["sender_upi"].(string)
	receiverUPI, ok2 := innerPayload["receiver_upi"].(string)
	amount, ok3 := innerPayload["amount"].(float64)

	if !ok1 || !ok2 || !ok3 || senderUPI == "" || receiverUPI == "" || amount <= 0 {
		respond(w, false, "invalid payment details in payload", http.StatusBadRequest)
		return
	}

	err = s.accounts.Transfer(senderUPI, receiverUPI, amount)
	if err != nil {
		respond(w, false, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.idempotency.MarkAsProcessed(packet.TransactionID)
	if err != nil {
		log.Println("WARNING: could not mark transaction as processed:", packet.TransactionID)
	}

	respond(w, true, "payment successful", http.StatusOK)
}

func (s *BankServer) handleBalance(w http.ResponseWriter, r *http.Request) {
	upi := r.URL.Query().Get("upi")
	if upi == "" {
		http.Error(w, "upi parameter required", http.StatusBadRequest)
		return
	}

	balance, err := s.accounts.GetBalance(upi)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"upi":     upi,
		"balance": balance,
	})
}

func respond(w http.ResponseWriter, success bool, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(PaymentResponse{
		Success: success,
		Message: message,
	})
}