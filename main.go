package main

import (
	"log"
	"net"
	"os"

	"wymux/pkg/pipeline"
)

func main() {
	log.Println("Starting WyMux Proxy Service...")

	// Default port for the Wyoming proxy service
	port := "10400"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	listener, err := net.Listen("tcp", "0.0.0.0:"+port)
	if err != nil {
		log.Fatalf("Failed to bind on local port %s: %v", port, err)
	}

	log.Printf("Listening for Wyoming connections on 0.0.0.0:%s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go pipeline.HandleConnection(conn)
	}
}
