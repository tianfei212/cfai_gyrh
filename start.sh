#!/bin/bash

# Define the target port range (Vite default is 3000, but may auto-increment)
PORT_START=3400
PORT_END=3405

echo "=========================================="
echo "   Starting GYRH Application Deployment   "
echo "=========================================="

# 1. Activate Conda Environment
echo "[1/3] Activating Conda environment 'gyrh'..."
# Source conda.sh to ensure 'conda' command and 'activate' work in script
if [ -f "/home/ubuntu/miniconda3/etc/profile.d/conda.sh" ]; then
    source "/home/ubuntu/miniconda3/etc/profile.d/conda.sh"
else
    echo "Error: Conda profile not found at /home/ubuntu/miniconda3/etc/profile.d/conda.sh"
    exit 1
fi

conda activate gyrh
if [ $? -ne 0 ]; then
    echo "Error: Failed to activate conda environment 'gyrh'"
    exit 1
fi
echo "      Environment activated: $(conda info --envs | grep '*' | awk '{print $1}')"

# 2. Check and Kill Existing Processes on Ports
echo "[2/3] Checking for existing processes on ports $PORT_START-$PORT_END..."
PIDS=$(lsof -t -i:$PORT_START-$PORT_END -sTCP:LISTEN)

if [ -n "$PIDS" ]; then
    echo "      Found existing processes: $PIDS"
    echo "      Stopping processes..."
    kill -9 $PIDS
    sleep 2 # Wait for OS to release ports
    echo "      Processes stopped and ports released."
else
    echo "      No conflicting processes found."
fi

# 3. Start Application
echo "[3/3] Starting application..."
TEMP_LOG="startup_buffer.log"
rm -f "$TEMP_LOG"

# Run in background, redirect output to temp log to capture URL
nohup npm run dev -- --host 0.0.0.0 > "$TEMP_LOG" 2>&1 &

# Get the PID of the last background command
NEW_PID=$!
echo "      Application started with PID: $NEW_PID"
echo "      Waiting for server initialization..."

# Wait for URL to appear in the log (max 10 seconds)
URL=""
for i in {1..10}; do
    if [ -f "$TEMP_LOG" ]; then
        # Look for "http://localhost:PORT/" pattern
        # Vite usually outputs: "Local:   http://localhost:3000/"
        URL=$(grep -o "http://localhost:[0-9]*/" "$TEMP_LOG" | head -n 1)
        if [ -n "$URL" ]; then
            break
        fi
    fi
    sleep 1
done

# Fallback if detection fails
if [ -z "$URL" ]; then
    URL="http://localhost:3000/ (Check logs if unreachable)"
fi

echo "=========================================="
echo "Success! Application is running in background."
echo "Access URL: $URL"
echo "Logs are being written to the 'logs/' directory."

# Clean up temp log
rm -f "$TEMP_LOG"
