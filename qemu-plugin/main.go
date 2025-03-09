package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"syscall"

	"github.com/lima-vm/lima/pkg/plugin"
)

type QemuPlugin struct{}

// Start handles the start request from the client
func (q *QemuPlugin) Start(args plugin.StartArgs, reply *bool) error {
	log.Printf("[QemuPlugin] Starting VM: %s", args.InstanceName)
	*reply = true
	return nil
}

func runServer(address string) error {
	plugin := &QemuPlugin{}

	// Register RPC service
	if err := rpc.Register(plugin); err != nil {
		return fmt.Errorf("RPC register error: %w", err)
	}

	// Start listening
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	defer listener.Close()

	log.Printf("[QemuPlugin] Listening on %s", address)

	// Handle shutdown signals
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("[QemuPlugin] Shutting down...")
		listener.Close()
		os.Exit(0)
	}()

	// Accept and serve connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		go rpc.ServeConn(conn)
	}
}

func main() {
	if err := runServer("127.0.0.1:9991"); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
