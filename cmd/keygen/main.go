package main

import (
	"fmt"
	"github.com/Amitesh-exp/NoInternetUpi/crypto"
)

func main() {
	err := crypto.GenerateAndSaveKeys("bank_private.pem", "bank_public.pem")
	if err != nil {
		fmt.Println("Error generating keys:", err)
		return
	}
	fmt.Println("Keys generated successfully")
	fmt.Println("bank_private.pem — keep this secret, only bank uses it")
	fmt.Println("bank_public.pem — share this with all nodes")
}