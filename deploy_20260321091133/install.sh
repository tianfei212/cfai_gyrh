#!/bin/bash
set -e
if [ -d node_modules ] && node -e "require.resolve('express')" >/dev/null 2>&1; then
  echo "Dependencies already installed."
else
  if [ -f package-lock.json ]; then
    npm ci --omit=dev --no-audit --no-fund
  else
    npm install --omit=dev --no-audit --no-fund
  fi
fi
echo "Installation complete. Run './start.sh' to launch with PM2."
