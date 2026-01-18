#!/bin/bash

# Script to restart the poc-finance server
# Usage: ./scripts/restart-server.sh

set -e

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_DIR"

echo "=== POC-Finance Server Restart Script ==="

# Find and kill existing server process on port 8080
echo "Checking for existing server on port 8080..."
PID=$(lsof -ti:8080 2>/dev/null || true)

if [ -n "$PID" ]; then
    echo "Found server running with PID: $PID"
    echo "Stopping server..."
    kill -9 $PID 2>/dev/null || true
    sleep 1
    echo "Server stopped."
else
    echo "No existing server found."
fi

# Rebuild the application
echo "Building application..."
go build -o bin/server ./cmd/server

if [ $? -eq 0 ]; then
    echo "Build successful!"
else
    echo "Build failed!"
    exit 1
fi

# Start the server
echo "Starting server..."
./bin/server &
SERVER_PID=$!

# Wait a moment for server to start
sleep 2

# Check if server is running
if kill -0 $SERVER_PID 2>/dev/null; then
    echo "=== Server started successfully ==="
    echo "Server PID: $SERVER_PID"
    echo "URL: http://localhost:8080"
    echo ""
    echo "To stop the server, run: kill $SERVER_PID"
else
    echo "Server failed to start!"
    exit 1
fi
