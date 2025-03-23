#!/bin/bash
set -eo pipefail

# Configuration
BINARY_NAME="dingus-aid"
OUTPUT_DIR="./output"
INSTALL_DIR="/usr/local/bin"
LOG_FILE="logs/setup_$(date +%Y%m%d_%H%M%S).log"

# Function for colorized logging
log() {
    local level=$1
    local message=$2
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    case "$level" in
        "INFO")
            # Green text
            echo -e "\033[0;32m[INFO]\033[0m $timestamp - $message"
            echo "[INFO] $timestamp - $message"
            ;;
        "WARN")
            # Yellow text
            echo -e "\033[0;33m[WARN]\033[0m $timestamp - $message"
            echo "[WARN] $timestamp - $message"
            ;;
        "ERROR")
            # Red text
            echo -e "\033[0;31m[ERROR]\033[0m $timestamp - $message"
            echo "[ERROR] $timestamp - $message"
            ;;
    esac
}

cleanup() {
    log "INFO" "Cleaning up temporary files..."
    # Add any cleanup tasks here if needed
}

handle_error() {
    log "ERROR" "An error occurred at line $1"
    cleanup
    exit 1
}

# Set up error handling
trap 'handle_error $LINENO' ERR

# Introduction
log "INFO" "Starting $BINARY_NAME installation script"
log "INFO" "Logs will be saved to $LOG_FILE"

# Step 1: Ensure output directory exists
if [ ! -d "$OUTPUT_DIR" ]; then
    log "INFO" "Creating output directory at $OUTPUT_DIR"
    mkdir -p "$OUTPUT_DIR" || {
        log "ERROR" "Failed to create output directory"
        exit 1
    }
else
    log "INFO" "Output directory already exists"
fi

# Step 2: Check if Docker is installed
if ! command -v docker &>/dev/null; then
    log "ERROR" "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker compose &>/dev/null; then
    log "ERROR" "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

# Step 3: Build the Docker image for dingus-aid
log "INFO" "Building the Docker image..."
if docker compose up --build --abort-on-container-exit; then
    log "INFO" "Docker build completed successfully"
else
    log "ERROR" "Docker build failed"
    exit 1
fi

# Step 4: Check if the binary exists locally
if [ ! -f "$OUTPUT_DIR/$BINARY_NAME" ]; then
    log "ERROR" "$BINARY_NAME binary not found in $OUTPUT_DIR"
    log "WARN" "Docker build may have failed to generate the binary"
    exit 1
fi

# Step 5: Make the binary executable
log "INFO" "Making the binary executable..."
chmod +x "$OUTPUT_DIR/$BINARY_NAME" || {
    log "ERROR" "Failed to make binary executable"
    exit 1
}

# Step 6: Check if we have permissions to write to install directory
if [ ! -w "$INSTALL_DIR" ]; then
    log "WARN" "You don't have write permissions for $INSTALL_DIR"
    log "INFO" "Attempting to use sudo for installation"
    
    # Attempt with sudo
    if ! command -v sudo &>/dev/null; then
        log "ERROR" "sudo command not found. Please run this script with root privileges."
        exit 1
    fi
    
    # Check if the binary already exists in the target location
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        log "WARN" "$BINARY_NAME already exists in $INSTALL_DIR. Creating backup..."
        sudo mv "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.backup" || {
            log "ERROR" "Failed to create backup of existing binary"
            exit 1
        }
        log "INFO" "Backup created at $INSTALL_DIR/${BINARY_NAME}.backup"
    fi
    
    # Copy the binary
    sudo cp "$OUTPUT_DIR/$BINARY_NAME" "$INSTALL_DIR/" || {
        log "ERROR" "Failed to copy binary to $INSTALL_DIR using sudo"
        exit 1
    }
    log "INFO" "Binary copied to $INSTALL_DIR successfully"
else
    # Check if the binary already exists in the target location
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        log "WARN" "$BINARY_NAME already exists in $INSTALL_DIR. Creating backup..."
        mv "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/${BINARY_NAME}.backup" || {
            log "ERROR" "Failed to create backup of existing binary"
            exit 1
        }
        log "INFO" "Backup created at $INSTALL_DIR/${BINARY_NAME}.backup"
    fi
    
    # Copy the binary instead of moving to preserve the original
    cp "$OUTPUT_DIR/$BINARY_NAME" "$INSTALL_DIR/" || {
        log "ERROR" "Failed to copy binary to $INSTALL_DIR"
        exit 1
    }
    log "INFO" "Binary copied to $INSTALL_DIR successfully"
fi

# Step 7: Confirm the binary is accessible
log "INFO" "Verifying installation..."
if command -v "$BINARY_NAME" &>/dev/null; then
    log "INFO" "$BINARY_NAME is now available in PATH"
    
    # Verify the binary actually works
    log "INFO" "Testing binary functionality..."
    if "$BINARY_NAME" --version &>/dev/null || "$BINARY_NAME" -v &>/dev/null; then
        log "INFO" "Binary test successful"
    else
        log "WARN" "Binary exists but returned non-zero exit code during testing"
    fi
else
    log "ERROR" "$BINARY_NAME command not found after installation"
    log "WARN" "You may need to add $INSTALL_DIR to your PATH or restart your terminal"
    exit 1
fi

# Step 8: Done!
log "INFO" "Setup complete! You can now use '$BINARY_NAME' from anywhere."
log "INFO" "Installation log saved to $LOG_FILE"