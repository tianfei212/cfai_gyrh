#!/bin/bash

echo "Starting Deployment Package Generation..."

# 1. Install Dependencies
echo "Installing dependencies..."
npm install

# 2. Build Frontend
echo "Building frontend..."
npm run build

# 3. Create Deploy Directory
echo "Creating deploy directory..."
rm -rf deploy
mkdir deploy

# 4. Copy Files
echo "Copying files..."
cp -r dist deploy/
cp server.js deploy/
cp package.json deploy/
cp package-lock.json deploy/ 2>/dev/null || true
cp ecosystem.config.js deploy/
cp prompts.json deploy/
cp siteConfig.json deploy/
# cp .env.local deploy/ 2>/dev/null || echo "No .env.local found, skipping..."
# cp .env deploy/ 2>/dev/null || echo "No .env found, skipping..."

# Create logs and old_pic directories
mkdir -p deploy/logs
mkdir -p deploy/old_pic

echo "Installing production dependencies into deploy package..."
if [ -f deploy/package-lock.json ]; then
  (cd deploy && npm ci --omit=dev --no-audit --no-fund)
else
  (cd deploy && npm install --omit=dev --no-audit --no-fund)
fi

# 5. Create Install Script for Server
echo "Creating install script..."
cat > deploy/install.sh << 'EOF'
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
EOF
chmod +x deploy/install.sh

# 6. Create Start/Stop Scripts
echo "Creating control scripts..."

# start.sh
cat > deploy/start.sh << 'EOF'
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
EOF
chmod +x deploy/start.sh

# stop.sh
cat > deploy/stop.sh << 'EOF'
#!/bin/bash
echo "Stopping application..."
pm2 stop ecosystem.config.js
echo "Application stopped."
EOF
chmod +x deploy/stop.sh

echo "Deployment package ready in 'deploy/' folder."

# 7. Create ZIP Package
echo "Creating ZIP package..."
TIMESTAMP=$(date +"%Y%m%d%H%M%S")
ZIP_NAME="deploy_${TIMESTAMP}.zip"

# Check if zip command exists
if command -v zip &> /dev/null; then
    cd deploy
    zip -r "../${ZIP_NAME}" .
    cd ..
    echo "Successfully created deployment package: ${ZIP_NAME}"
else
    echo "Error: 'zip' command not found. Please install zip or manually compress the deploy folder."
fi
