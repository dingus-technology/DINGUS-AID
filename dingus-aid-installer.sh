#!/bin/bash

# Step 1: Ensure output directory exists
mkdir -p output/

# Step 2: Build the Docker image for dingus-aid
echo "Building the Docker image..."
docker compose up --build --abort-on-container-exit

# Step 3: Check if the binary exists locally
if [ ! -f "./output/dingus-aid" ]; then
    echo "Error: dingus-aid binary not found in ./output."
    exit 1
fi

# Step 4: Make the binary executable
echo "Making the binary executable..."
chmod +x ./output/dingus-aid

# Step 5: Move the binary to /usr/local/bin for global use
echo "Moving the binary to /usr/local/bin..."
mv ./output/dingus-aid /usr/local/bin/

# Step 6: Confirm the binary is accessible
if ! command -v dingus-aid &>/dev/null; then
    echo "Error: dingus-aid command not found after installation."
    exit 1
fi

# Step 7: Done!
echo "Setup complete! You can now use 'dingus-aid' from anywhere."
