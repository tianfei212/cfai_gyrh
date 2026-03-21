#!/bin/bash
echo "Stopping application..."
pm2 stop ecosystem.config.js
echo "Application stopped."
