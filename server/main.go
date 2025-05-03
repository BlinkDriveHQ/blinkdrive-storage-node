package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	pb "storagenode/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort        = 50051
	defaultStoragePath = "./storage"
)

func main() {
	port := flag.Int("port", defaultPort, "The server port")
	nodeID := flag.String("node-id", "", "Unique ID for this storage node")
	storagePath := flag.String("storage", defaultStoragePath, "Path to store files")
	flag.Parse()

	if *nodeID == "" {
		hostname, err := os.Hostname()
		if err == nil {
			*nodeID = fmt.Sprintf("node-%s", hostname)
		} else {
			*nodeID = fmt.Sprintf("node-%d", time.Now().Unix())
		}
	}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Configure gRPC server with larger message size limits to handle base64 data
	// Base64 encoding increases size by approximately 33%
	maxMsgSize := 100 * 1024 * 1024 // 100MB
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	)

	storageNode := NewStorageNodeServer(*nodeID, *storagePath)
	pb.RegisterStorageNodeServer(server, storageNode)

	// Register reflection service for debugging tools
	reflection.Register(server)

	log.Printf("Base64-compatible Storage Node %s starting on port %d", *nodeID, *port)
	log.Printf("Storage path: %s", *storagePath)
	log.Printf("Maximum message size: %d MB", maxMsgSize/(1024*1024))

	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
