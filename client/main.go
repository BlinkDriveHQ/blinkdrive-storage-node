package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	pb "storagenode/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	serverAddress = "localhost:50051"
	chunkSize     = 768 * 1024 // 768KB chunks (accounts for Base64 expansion)
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	conn, err := grpc.NewClient(serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(100*1024*1024),
		))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewStorageNodeClient(conn)

	cmd := os.Args[1]

	switch cmd {
	case "upload":
		if len(os.Args) < 3 {
			printUsage()
			return
		}
		uploadFile(client, os.Args[2])
	case "download":
		if len(os.Args) < 3 {
			printUsage()
			return
		}
		downloadFile(client, os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			printUsage()
			return
		}
		deleteFile(client, os.Args[2])
	case "status":
		getStatus(client)
	default:
		printUsage()
	}
}

func uploadFile(client pb.StorageNodeClient, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Could not open file: %v", err)
	}
	defer file.Close()

	stream, err := client.UploadFile(context.Background())
	if err != nil {
		log.Fatalf("Error creating upload stream: %v", err)
	}

	buffer := make([]byte, chunkSize)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		// Encode to Base64 before sending
		encoded := base64.StdEncoding.EncodeToString(buffer[:n])
		if err := stream.Send(&pb.FileChunk{
			Content: []byte(encoded),
		}); err != nil {
			log.Fatalf("Error sending chunk: %v", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Upload failed: %v", err)
	}

	log.Printf("Upload successful. File ID: %s", resp.FileId)
}

func downloadFile(client pb.StorageNodeClient, fileID string) {
	req := &pb.FileRequest{FileId: fileID}
	stream, err := client.DownloadFile(context.Background(), req)
	if err != nil {
		log.Fatalf("Download failed: %v", err)
	}

	outputPath := fmt.Sprintf("downloaded_%s_%d", fileID, time.Now().Unix())
	file, err := os.Create(outputPath)
	if err != nil {
		log.Fatalf("Could not create file: %v", err)
	}
	defer file.Close()

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Error receiving chunk: %v", err)
		}

		// Write raw Base64 data to file (as received from server)
		if _, err := file.Write(chunk.Content); err != nil {
			log.Fatalf("Error writing chunk: %v", err)
		}
	}

	log.Printf("File saved as: %s (contains Base64 data)", outputPath)
}

func deleteFile(client pb.StorageNodeClient, fileID string) {
	resp, err := client.DeleteFile(context.Background(), &pb.FileRequest{FileId: fileID})
	if err != nil {
		log.Fatalf("Delete failed: %v", err)
	}

	if resp.Success {
		log.Println("File deleted successfully")
	} else {
		log.Printf("Delete failed: %s", resp.Message)
	}
}

func getStatus(client pb.StorageNodeClient) {
	resp, err := client.GetNodeStatus(context.Background(), &pb.StatusRequest{})
	if err != nil {
		log.Fatalf("Status check failed: %v", err)
	}

	fmt.Printf("Node Status:\n")
	fmt.Printf("ID: %s\n", resp.NodeId)
	fmt.Printf("Files Stored: %d\n", resp.FilesStored)
	fmt.Printf("Available Space: %.2f MB\n", resp.DiskSpaceAvailable/(1024*1024))
	fmt.Printf("Healthy: %v\n", resp.Healthy)
}

func printUsage() {
	fmt.Println("Usage: client <command> [arguments]")
	fmt.Println("Commands:")
	fmt.Println("  upload <file>       - Upload a file")
	fmt.Println("  download <file-id>  - Download a file")
	fmt.Println("  delete <file-id>    - Delete a file")
	fmt.Println("  status              - Get server status")
}
