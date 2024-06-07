#!/bin/bash

set -e

# the directory containing the source files
SOURCE_DIR=$(pwd)

# paths for the binary and eBPF object file
BINARY_NAME="packet-dropper"
EBPF_FILE_NAME="packet_drop_kern.o"
INSTALL_DIR="/usr/local/bin"
EBPF_INSTALL_DIR="/etc/packet-dropper"

# Install prerequisites
echo "Installing prerequisites..."
sudo apt update
sudo apt install -y clang llvm golang

# Compile the eBPF program
echo "Compiling the eBPF program..."
clang -O2 -g -Wall -target bpf -c "$SOURCE_DIR/packet_drop.c" -o "$SOURCE_DIR/$EBPF_FILE_NAME"

# Download Go dependencies
echo "Downloading Go dependencies..."
cd "$SOURCE_DIR"
GO111MODULE=on go mod download

# Build the Go program
echo "Building the Go program..."
go build -o "$BINARY_NAME" packet_drop.go

# Move the binary to /usr/local/bin
echo "Installing the packet-dropper binary..."
sudo mv "$BINARY_NAME" "$INSTALL_DIR/"

# Create the directory for the eBPF object file if it doesn't exist
echo "Setting up the eBPF object file..."
sudo mkdir -p "$EBPF_INSTALL_DIR"

# Move the eBPF object file to the created directory
sudo mv "$SOURCE_DIR/$EBPF_FILE_NAME" "$EBPF_INSTALL_DIR/"

# Make sure the eBPF object file is accessible
sudo chmod 644 "$EBPF_INSTALL_DIR/$EBPF_FILE_NAME"

echo "Installation complete."
echo "You can now run the packet-dropper with the following command:"
echo "  :: sudo packet-dropper --interface <your-network-interface> --port <port-number>"
echo "To list available network interfaces, use:"
echo "  :: packet-dropper --list-interfaces"
