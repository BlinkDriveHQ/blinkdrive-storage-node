package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/windows"

	pb "storagenode/proto"
)

// StorageNodeServer implements the gRPC StorageNode service
type StorageNodeServer struct {
	pb.UnimplementedStorageNodeServer
	nodeID      string
	storagePath string
	mu          sync.Mutex
	files       map[string]fileMetadata
}

type fileMetadata struct {
	filePath   string
	size       int64
	uploadTime time.Time
	isBase64   bool // Flag to indicate the file is stored in base64 format
}

// NewStorageNodeServer creates a new storage node server instance
func NewStorageNodeServer(nodeID string, storagePath string) *StorageNodeServer {
	// Create storage directory if it doesn't exist
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	log.Printf("Storage node initialized with base64 file support")
	return &StorageNodeServer{
		nodeID:      nodeID,
		storagePath: storagePath,
		files:       make(map[string]fileMetadata),
	}
}

// UploadFile implements the RPC method to handle file uploads
// It expects the incoming file chunks to already be base64 encoded
func (s *StorageNodeServer) UploadFile(stream pb.StorageNode_UploadFileServer) error {
	// Initialize variables to store the file
	var fileID string
	var filePath string
	var file *os.File
	var firstChunk = true

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// End of file reached
			if file != nil {
				file.Close()
			}

			s.mu.Lock()
			fileInfo, err := os.Stat(filePath)
			if err == nil {
				s.files[fileID] = fileMetadata{
					filePath:   filePath,
					size:       fileInfo.Size(),
					uploadTime: time.Now(),
					isBase64:   true, // Mark as base64
				}
				log.Printf("File %s stored as base64, size: %d bytes", fileID, fileInfo.Size())
			}
			s.mu.Unlock()

			return stream.SendAndClose(&pb.UploadResponse{
				Success: true,
				FileId:  fileID,
				Message: "Base64 file uploaded successfully",
			})
		}
		if err != nil {
			return fmt.Errorf("error receiving chunk: %v", err)
		}

		// Handle the first chunk
		if firstChunk {
			fileID = chunk.FileId
			if fileID == "" {
				// Generate a file ID if not provided
				hash := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", s.nodeID, time.Now().UnixNano())))
				fileID = hex.EncodeToString(hash[:])
			}

			filePath = filepath.Join(s.storagePath, fileID+".b64") // Add .b64 extension for clarity
			var err error
			file, err = os.Create(filePath)
			if err != nil {
				return fmt.Errorf("failed to create file: %v", err)
			}
			firstChunk = false
			log.Printf("Started receiving base64 file with ID: %s", fileID)
		}

		// Write the chunk directly to the file (already base64 encoded)
		if _, err := file.Write(chunk.Content); err != nil {
			file.Close()
			return fmt.Errorf("failed to write chunk: %v", err)
		}
	}
}

// DownloadFile implements the RPC method to download a file
// Returns the file in its stored base64 format
func (s *StorageNodeServer) DownloadFile(req *pb.FileRequest, stream pb.StorageNode_DownloadFileServer) error {
	fileID := req.FileId

	s.mu.Lock()
	metadata, exists := s.files[fileID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("file not found: %s", fileID)
	}

	file, err := os.Open(metadata.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 1024*1024) // 1MB chunks
	chunkNumber := 0

	log.Printf("Streaming base64 file %s to client", fileID)

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading file: %v", err)
		}

		chunkNumber++

		if err := stream.Send(&pb.FileChunk{
			FileId:      fileID,
			Content:     buffer[:n],
			ChunkNumber: int32(chunkNumber),
		}); err != nil {
			return fmt.Errorf("error sending chunk: %v", err)
		}
	}

	log.Printf("Completed sending base64 file %s, total chunks: %d", fileID, chunkNumber)
	return nil
}

// DeleteFile implements the RPC method to delete a file
func (s *StorageNodeServer) DeleteFile(ctx context.Context, req *pb.FileRequest) (*pb.DeleteResponse, error) {
	fileID := req.FileId

	s.mu.Lock()
	metadata, exists := s.files[fileID]
	if !exists {
		s.mu.Unlock()
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("File not found: %s", fileID),
		}, nil
	}

	filePath := metadata.filePath
	delete(s.files, fileID)
	s.mu.Unlock()

	if err := os.Remove(filePath); err != nil {
		return &pb.DeleteResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to delete file: %v", err),
		}, nil
	}

	log.Printf("Deleted base64 file: %s", fileID)

	return &pb.DeleteResponse{
		Success: true,
		Message: "File deleted successfully",
	}, nil
}

// / GetNodeStatus implements the RPC method to get the status of the storage node
func (s *StorageNodeServer) GetNodeStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	var freeBytes, totalBytes, availableBytes uint64

	// Use Windows-specific API since you're developing on Windows
	err := windows.GetDiskFreeSpaceEx(
		windows.StringToUTF16Ptr(s.storagePath),
		&freeBytes,
		&totalBytes,
		&availableBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk space: %v", err)
	}

	s.mu.Lock()
	fileCount := int32(len(s.files))
	s.mu.Unlock()

	return &pb.StatusResponse{
		DiskSpaceAvailable: float64(availableBytes),
		FilesStored:        fileCount,
		NodeId:             s.nodeID,
		Healthy:            true,
	}, nil
}
