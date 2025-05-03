# BlinkDrive Storage Node Setup Guide

## Prerequisites

- Go 1.20+ installed
- Git Bash/MINGW64 (recommended) or PowerShell
- Admin privileges (for system configurations)

## 1. Environment Setup

```bash
# Install protoc (Protocol Buffer compiler)
curl -LO https://github.com/protocolbuffers/protobuf/releases/download/v24.4/protoc-24.4-win64.zip
unzip protoc-24.4-win64.zip -d C:\protoc
setx PATH "%PATH%;C:\protoc\bin" /M

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Clone repository (if applicable)
git clone https://github.com/your-repo/blinkdrive-storage-node.git
cd blinkdrive-storage-node

# Initialize modules
go mod init blinkdrive-storage-node
go mod tidy

# From project root
protoc --go_out=. --go-grpc_out=. proto/storage.proto

# Standard build (produces .exe)
go build -o storage-node ./server

# Basic execution
./storage-node --port 50051 --node-id node1 --storage ./storage_node1
