package node

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Amitesh-exp/NoInternetUpi/models"
)

const (
	ListenPort = 9999
)

var KnownNodes = []string{
	"127.0.0.1:9998",
	"127.0.0.1:9997",
	"127.0.0.1:9996",
}

func Broadcast(packet models.PaymentPacket, myPort int) error {
	data, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	sent := 0
	for _, nodeAddr := range KnownNodes {
		addr, err := net.ResolveUDPAddr("udp", nodeAddr)
		if err != nil {
			continue
		}

		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			continue
		}

		_, err = conn.Write(data)
		conn.Close()
		if err != nil {
			continue
		}
		sent++
	}

	fmt.Printf("Packet broadcasted to %d nodes: %s\n", sent, packet.TransactionID)
	return nil
}

func Listen(port int, handler func(models.PaymentPacket)) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return fmt.Errorf("failed to resolve listen address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port %d: %w", port, err)
	}
	defer conn.Close()

	fmt.Printf("Listening for packets on port %d\n", port)

	buf := make([]byte, 65536)

	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP packet:", err)
			continue
		}

		var packet models.PaymentPacket
		err = json.Unmarshal(buf[:n], &packet)
		if err != nil {
			fmt.Println("Error deserializing packet:", err)
			continue
		}

		go handler(packet)
	}
}