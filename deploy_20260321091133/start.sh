#!/bin/bash
set -e
# Check if PM2 is installed
if ! command -v pm2 &> /dev/null; then
    echo "PM2 not found. Installing PM2 globally..."
    npm install -g pm2
fi

./install.sh

echo "Starting application with PM2..."
pm2 start ecosystem.config.js --update-env
pm2 save
echo "Application started."
