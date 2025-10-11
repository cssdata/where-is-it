#!/bin/bash

# Build the Go application
echo "Building Go application..."
go build -o where-is-it .

if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

echo "Build successful!"

# Create data directory if it doesn't exist
mkdir -p data

echo "Starting application on http://localhost:8080"
echo "Press Ctrl+C to stop"

# Run the application
./where-is-it