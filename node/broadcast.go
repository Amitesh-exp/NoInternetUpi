package node

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/Amitesh-exp/NoInternetUpi/models"
)

const (
	BroadcastPort = 9999
	BroadcastAddr = "255.255.255.255:9999"
)

// Broadcast sends a payment packet to all nodes on the LAN
// Uses UDP broadcast — no specific target, everyone receives it
func Broadcast(packet models.PaymentPacket) error {
	// Resolve the broadcast address
	addr, err := net.ResolveUDPAddr("udp", BroadcastAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	// Open a UDP connection
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to open UDP connection: %w", err)
	}
	defer conn.Close()

	// Serialize the packet to JSON bytes
	data, err := json.Marshal(packet)
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	// Send it — UDP fire and forget
	// We don't wait for confirmation, don't know who received it
	_, err = conn.Write(data)
	if err != nil {
		return fmt.Errorf("failed to broadcast packet: %w", err)
	}

	fmt.Println("Packet broadcasted:", packet.TransactionID)
	return nil
}

// Listen starts listening for incoming UDP packets from other nodes
// When a packet arrives, it's passed to the handler function
func Listen(handler func(models.PaymentPacket)) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", BroadcastPort))
	if err != nil {
		return fmt.Errorf("failed to resolve listen address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP port: %w", err)
	}
	defer conn.Close()

	fmt.Println("Listening for packets on UDP port", BroadcastPort)

	// Buffer to hold incoming data — 64KB max packet size
	buf := make([]byte, 65536)

	for {
		// Block until a packet arrives
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error reading UDP packet:", err)
			continue
		}

		// Deserialize the packet
		var packet models.PaymentPacket
		err = json.Unmarshal(buf[:n], &packet)
		if err != nil {
			fmt.Println("Error deserializing packet:", err)
			continue
		}

		// Pass to handler — node decides what to do with it
		go handler(packet)
	}
} 	