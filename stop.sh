#!/bin/bash

echo "Stopping GYRH Application..."

# Find process by port 3000 (default vite port) or by command
# Using lsof to find PID listening on port 3000-3005 (in case of port increments)
PIDS=$(lsof -t -i:3400-3405 -sTCP:LISTEN)

if [ -z "$PIDS" ]; then
  echo "No process found listening on ports 3400-3405."
else
  echo "Killing processes: $PIDS"
  kill -9 $PIDS
  echo "Stopped."
fi
