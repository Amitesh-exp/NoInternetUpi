package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Amitesh-exp/NoInternetUpi/models"
	"github.com/Amitesh-exp/NoInternetUpi/node"
)

func main() {
	args := os.Args[1:]
	if len(args) < 2 {
		fmt.Println("Usage: go run cmd/node/main.go <node-name> <has-internet>")
		fmt.Println("Example: go run cmd/node/main.go Alice true")
		fmt.Println("Example: go run cmd/node/main.go Bob false")
		os.Exit(1)
	}

	nodeName := args[0]
	hasInternet := args[1] == "true"
	bankURL := "http://localhost:8080"
	publicKeyPath := "bank_public.pem"

	fmt.Printf("Node %s started | Internet: %v\n", nodeName, hasInternet)
	fmt.Println("Listening for incoming packets from nearby nodes...")

	go node.Listen(func(packet models.PaymentPacket) {
		fmt.Printf("\n[%s] Received packet: %s\n", nodeName, packet.TransactionID)

		if time.Now().After(packet.TTL) {
			fmt.Printf("[%s] Packet expired — ignoring\n", nodeName)
			return
		}

		if !hasInternet {
			fmt.Printf("[%s] No internet — cannot forward\n", nodeName)
			return
		}

		fmt.Printf("[%s] Has internet — forwarding to bank\n", nodeName)
		err := node.ForwardToBank(packet, bankURL)
		if err != nil {
			fmt.Printf("[%s] Forward failed: %v\n", nodeName, err)
			return
		}
		fmt.Printf("[%s] Forward successful\n", nodeName)
	})

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\nOptions:")
		fmt.Println("1. Send payment")
		fmt.Println("2. Exit")
		fmt.Print("Choice: ")

		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			fmt.Print("Your UPI ID: ")
			scanner.Scan()
			senderUPI := strings.TrimSpace(scanner.Text())

			fmt.Print("Receiver UPI ID: ")
			scanner.Scan()
			receiverUPI := strings.TrimSpace(scanner.Text())

			fmt.Print("Amount: ")
			scanner.Scan()
			amountStr := strings.TrimSpace(scanner.Text())
			amount, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				fmt.Println("Invalid amount")
				continue
			}

			err = node.CreateAndBroadcastPayment(senderUPI, receiverUPI, amount, publicKeyPath, hasInternet, bankURL)
			if err != nil {
				fmt.Println("Payment failed:", err)
			}

		case "2":
			fmt.Println("Exiting...")
			os.Exit(0)

		default:
			fmt.Println("Invalid choice")
		}
	}
}