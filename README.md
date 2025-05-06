# BlinkDrive Storage Node

A Go-based gRPC server that provides file storage capabilities for the BlinkDrive Distributed File System.

## Overview

BlinkDrive Storage Node handles the physical storage of files within the distributed file system. It provides gRPC endpoints for uploading, downloading, and deleting files, with built-in support for base64 encoding/decoding.

## Features

- **gRPC Interface**: High-performance streaming file transfers
- **Base64 Compatibility**: Built-in support for base64-encoded file content
- **Health Monitoring**: Status endpoint for health checks and capacity reporting
- **Replication Support**: Works in a cluster for redundant file storage
- **Simple File Management**: File storage on the local filesystem with unique IDs

## Requirements

- Go 1.20 or higher
- Protobuf compiler (protoc) and gRPC tools
- Sufficient disk space for file storage

## Setup and Configuration

1. Clone the repository:

   ```bash
   git clone https://github.com/BlinkDriveHQ/blinkdrive-storage-node.git
   cd blinkdrive-storage-node
   ```

2. Install dependencies:

   ```bash
   go mod tidy
   ```

3. Compile the protobuf definitions:

   ```bash
   protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/storage.proto
   ```

4. Build the server:

   ```bash
   go build -o storage-node ./server
   ```

5. Run the server:

   ```bash
   ./storage-node --port 50051 --node-id node1 --storage ./storage_node1
   ./storage-node --port 50052 --node-id node2 --storage ./storage_node2
   ./storage-node --port 50053 --node-id node3 --storage ./storage_node3
   ./storage-node --port 50054 --node-id node4 --storage ./storage_node4
   ./storage-node --port 50055 --node-id node5 --storage ./storage_node5
   ```

## Client Usage (Testing)

1. Build the client:

    ```bash
    go build -o storage-client ./client
    ```

2. Client Execution:

    ```txt
    # Upload File
    ./storage-client upload test.txt
    ./storage-client upload dir/test.txt

    # Download File (replace with actual file ID)
    ./storage-client download file-id-here

    # Delete a file
    ./storage-client delete file-id-here

    # Get node status
    ./storage-client status
    ```

## Command Line Options

- `--port`: The gRPC server port (default: 50051)
- `--node-id`: Unique identifier for this storage node (default: auto-generated)
- `--storage`: Path to the storage directory (default: "./storage")

## API Reference

The gRPC service definition:

```protobuf
service StorageNode {
    rpc UploadFile(stream FileChunk) returns (UploadResponse);
    rpc DownloadFile(FileRequest) returns (stream FileChunk);
    rpc DeleteFile(FileRequest) returns (DeleteResponse);
    rpc GetNodeStatus(StatusRequest) returns (StatusResponse);
}
```

### Message Types

```protobuf
message FileChunk {
    string file_id = 1;
    bytes content = 2;
    int32 chunk_number = 3;
}

message FileRequest {
    string file_id = 1;
}

message UploadResponse {
    bool success = 1;
    string file_id = 2;
    string message = 3;
}

message DeleteResponse {
    bool success = 1;
    string message = 2;
}

message StatusRequest {}

message StatusResponse {
    double disk_space_available = 1;
    int32 files_stored = 2;
    string node_id = 3;
    bool healthy = 4;
}
```

## File Storage Structure

Files are stored in the configured storage directory with their file ID as the filename, with a `.b64` extension to indicate base64 encoding:

```md
/storage/
  └── 3f7c8d9e-1a2b-3c4d-5e6f-7g8h9i0j1k2l.b64
  └── a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6.b64
  └── ...
```

## Performance Considerations

- The server is configured with a 100MB message size limit to handle large files
- Files are streamed in chunks to manage memory usage
- Base64 encoding increases the storage size by approximately 33%

## Development

### Project Structure

```md
/
├── proto/
│   └── storage.proto          - gRPC service definition
├── server/
│   ├── main.go                - Server entry point
│   └── server.go              - StorageNode implementation
├── client/
│   └── client.go              - Example client implementation
├── go.mod                     - Go module definition
└── go.sum                     - Go dependency checksums
```

### Building and Testing

To build and run the server in development:

```bash
go run ./server/main.go --port 50051
```

To run tests:

```bash
go test ./...
```

## Deployment

For production deployment, consider:

1. Packaging as a systemd service
2. Setting up proper log rotation
3. Configuring disk monitoring
4. Implementing automated node recovery

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
