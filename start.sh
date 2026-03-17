#!/bin/bash
set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

PORT_START=3400
PORT_END=3405
CONDA_ENV_NAME="gryh"
LOG_DIR="$SCRIPT_DIR/logs"
STARTUP_LOG="$LOG_DIR/devserver.log"

echo "=========================================="
echo "   Starting GYRH Application Deployment   "
echo "=========================================="

echo "[1/3] Activating Conda environment '$CONDA_ENV_NAME'..."
CONDA_PROFILE=""
if [ -n "${CONDA_EXE:-}" ] && [ -f "$(dirname "$(dirname "$CONDA_EXE")")/etc/profile.d/conda.sh" ]; then
  CONDA_PROFILE="$(dirname "$(dirname "$CONDA_EXE")")/etc/profile.d/conda.sh"
elif [ -f "$HOME/miniconda3/etc/profile.d/conda.sh" ]; then
  CONDA_PROFILE="$HOME/miniconda3/etc/profile.d/conda.sh"
elif [ -f "$HOME/anaconda3/etc/profile.d/conda.sh" ]; then
  CONDA_PROFILE="$HOME/anaconda3/etc/profile.d/conda.sh"
fi

if [ -n "$CONDA_PROFILE" ]; then
  source "$CONDA_PROFILE"
  if conda activate "$CONDA_ENV_NAME" 2>/dev/null; then
    echo "      Environment activated: $(conda info --envs | awk '/\*/ {print $1}')"
  else
    echo "      Warning: Conda env '$CONDA_ENV_NAME' not found, continue with current shell."
  fi
else
  echo "      Warning: Conda profile not found, continue with current shell."
fi

echo "[2/3] Checking for existing processes on ports $PORT_START-$PORT_END..."
PIDS=""
for p in $(seq "$PORT_START" "$PORT_END"); do
  CUR_PIDS="$(lsof -t -iTCP:$p -sTCP:LISTEN 2>/dev/null || true)"
  if [ -n "$CUR_PIDS" ]; then
    PIDS="$PIDS $CUR_PIDS"
  fi
done
PIDS="$(echo "$PIDS" | xargs -n1 | sort -u | xargs)"

if [ -n "$PIDS" ]; then
  echo "      Found existing processes: $PIDS"
  echo "      Stopping processes..."
  kill -9 $PIDS
  sleep 2
  echo "      Processes stopped and ports released."
else
  echo "      No conflicting processes found."
fi

echo "[3/3] Starting application..."
mkdir -p "$LOG_DIR"
rm -f "$STARTUP_LOG"
nohup npm run dev -- --host 0.0.0.0 --port "$PORT_START" > "$STARTUP_LOG" 2>&1 &

NEW_PID=$!
echo "      Application started with PID: $NEW_PID"
echo "      Waiting for server initialization..."

URL=""
for i in {1..15}; do
  if [ -f "$STARTUP_LOG" ]; then
    URL="$(grep -Eo "http://(localhost|127\.0\.0\.1|0\.0\.0\.0):[0-9]+/" "$STARTUP_LOG" | head -n 1)"
    if [ -n "$URL" ]; then
      break
    fi
  fi
  sleep 1
done

if [ -z "$URL" ]; then
  URL="http://localhost:${PORT_START}/ (Check $STARTUP_LOG)"
fi

echo "=========================================="
echo "Success! Application is running in background."
echo "Access URL: $URL"
echo "Log file: $STARTUP_LOG"
