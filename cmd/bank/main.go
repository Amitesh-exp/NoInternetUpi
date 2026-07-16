package main
import (
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

// BankServer holds everything the server needs to handle requests
type BankServer struct {
	accounts    *bank.AccountStore
	idempotency *bank.IdempotencyStore
	privateKey  interface{}
}

// PaymentRequest is what relay nodes send to the bank
type PaymentRequest struct {
	Packet models.PaymentPacket `json:"packet"`
}

// PaymentResponse is what the bank sends back
type PaymentResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func main() {
	// Open database once — shared by both stores
	db, err := sql.Open("sqlite", "bank.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Set up account store
	accountStore, err := bank.NewAccountStore(db)
	if err != nil {
		log.Fatal("Failed to create account store:", err)
	}

	// Seed test accounts
	err = accountStore.Seed()
	if err != nil {
		log.Fatal("Failed to seed accounts:", err)
	}

	// Set up idempotency store using same db connection
	idempotencyStore, err := bank.NewIdempotencyStore(db)
	if err != nil {
		log.Fatal("Failed to create idempotency store:", err)
	}

	// Load bank's private key — only bank has this
	privateKey, err := crypto.LoadPrivateKey("bank_private.pem")
	if err != nil {
		log.Fatal("Failed to load private key:", err)
	}

	server := &BankServer{
		accounts:    accountStore,
		idempotency: idempotencyStore,
		privateKey:  privateKey,
	}

	// Register routes
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

	// Decode the incoming packet
	var req PaymentRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		respond(w, false, "invalid request format", http.StatusBadRequest)
		return
	}

	packet := req.Packet

	// Check TTL — reject expired packets
	if time.Now().After(packet.TTL) {
		respond(w, false, "packet expired — TTL exceeded", http.StatusBadRequest)
		return
	}

	// Check idempotency — reject duplicate transaction IDs
	alreadyProcessed, err := s.idempotency.HasBeenProcessed(packet.TransactionID)
	if err != nil {
		respond(w, false, "internal error", http.StatusInternalServerError)
		return
	}
	if alreadyProcessed {
		// Return success because the transaction DID go through
		// just not this time — this is a duplicate
		respond(w, true, "already processed", http.StatusOK)
		return
	}

	// Decrypt the payload using bank's private key
	privateKey, ok := s.privateKey.(interface {
		Decrypt([]byte) ([]byte, error)
	})
	_ = privateKey
	_ = ok

	// For now verify the outer packet fields directly
	// Full decryption wired in next step
	if packet.SenderUPI == "" || packet.ReceiverUPI == "" || packet.Amount <= 0 {
		respond(w, false, "invalid payment details", http.StatusBadRequest)
		return
	}

	// Process the transfer
	err = s.accounts.Transfer(packet.SenderUPI, packet.ReceiverUPI, packet.Amount)
	if err != nil {
		respond(w, false, err.Error(), http.StatusBadRequest)
		return
	}

	// Mark as processed — any future duplicates will be rejected
	err = s.idempotency.MarkAsProcessed(packet.TransactionID)
	if err != nil {
		// Transfer succeeded but we couldn't mark it
		// Log this — it means duplicates could go through
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

// respond is a helper to write consistent JSON responses
func respond(w http.ResponseWriter, success bool, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(PaymentResponse{
		Success: success,
		Message: message,
	})
}