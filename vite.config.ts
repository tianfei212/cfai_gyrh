import path from 'path';
import fs from 'fs';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig(({ mode }) => {
    const env = loadEnv(mode, '.', '');

    // Setup Logger
    const logsDir = path.join(__dirname, 'logs');
    if (!fs.existsSync(logsDir)) {
      fs.mkdirSync(logsDir, { recursive: true });
    }
    
    // Create a new log file for this session
    const now = new Date();
    // Use YYYYMMDD format
    const year = now.getFullYear();
    const month = String(now.getMonth() + 1).padStart(2, '0');
    const day = String(now.getDate()).padStart(2, '0');
    const timestampStr = `${year}${month}${day}`;
    const logFilePath = path.join(logsDir, `${timestampStr}.log`);
    
    const logToFile = (message: string) => {
      const time = new Date().toISOString();
      const logLine = `[${time}] ${message}\n`;
      try {
        fs.appendFileSync(logFilePath, logLine);
        process.stdout.write(logLine); // Also write to console
      } catch (e) {
        console.error("Failed to write log:", e);
      }
    };

    // Log startup
    logToFile("System Started. Vite Server Initializing...");
    
    // Custom plugin for file system operations
    const fileSystemPlugin = () => ({
      name: 'file-system-api',
      configureServer(server) {
        server.middlewares.use(async (req, res, next) => {
          // Logger API
          if (req.url === '/api/log' && req.method === 'POST') {
             const chunks = [];
             req.on('data', chunk => chunks.push(chunk));
             req.on('end', () => {
               try {
                 const body = JSON.parse(Buffer.concat(chunks).toString());
                 const { message, details, level = 'INFO' } = body;
                 
                 let logMsg = `[${level}] ${message}`;
                 if (details) {
                   logMsg += ` - ${JSON.stringify(details)}`;
                 }
                 logToFile(logMsg);
                 
                 res.statusCode = 200;
                 res.end(JSON.stringify({ success: true }));
               } catch (err) {
                 console.error('Error processing log:', err);
                 res.statusCode = 500;
                 res.end(JSON.stringify({ error: 'Failed to process log' }));
               }
             });
             return;
          }

          // API to save image
          if (req.url === '/api/save-image' && req.method === 'POST') {
            const chunks = [];
            req.on('data', chunk => chunks.push(chunk));
            req.on('end', () => {
              try {
                const body = JSON.parse(Buffer.concat(chunks).toString());
                const { name, data } = body;
                
                if (!name || !data) {
                  logToFile(`[ERROR] Save Image Failed: Missing name or data`);
                  res.statusCode = 400;
                  res.end(JSON.stringify({ error: 'Missing name or data' }));
                  return;
                }

                // Ensure directory exists
                const dir = path.join(__dirname, 'old_pic');
                if (!fs.existsSync(dir)) {
                  fs.mkdirSync(dir);
                }

                // Remove header if present (data:image/png;base64,...)
                const base64Data = data.replace(/^data:image\/\w+;base64,/, "");
                const buffer = Buffer.from(base64Data, 'base64');
                
                const filePath = path.join(dir, name);
                fs.writeFileSync(filePath, buffer);
                
                logToFile(`[INFO] Image Saved: ${name}`);

                res.setHeader('Content-Type', 'application/json');
                res.end(JSON.stringify({ success: true, path: filePath }));
              } catch (err) {
                logToFile(`[ERROR] Save Image Failed: ${err.message}`);
                res.statusCode = 500;
                res.end(JSON.stringify({ error: 'Failed to save file' }));
              }
            });
            return;
          }

          // API to list images
          if (req.url === '/api/list-images' && req.method === 'GET') {
            try {
              const dir = path.join(__dirname, 'old_pic');
              if (!fs.existsSync(dir)) {
                res.setHeader('Content-Type', 'application/json');
                res.end(JSON.stringify([]));
                return;
              }

              const files = fs.readdirSync(dir)
                .filter(file => /\.(png|jpg|jpeg|webp)$/i.test(file))
                .map(file => {
                  const stats = fs.statSync(path.join(dir, file));
                  return {
                    name: file,
                    timestamp: stats.mtimeMs,
                    // We'll serve files via a separate route or static serving
                    url: `/old_pic/${file}` 
                  };
                })
                .sort((a, b) => b.timestamp - a.timestamp); // Newest first

              logToFile(`[INFO] List Images: Found ${files.length} files`);
              
              res.setHeader('Content-Type', 'application/json');
              res.end(JSON.stringify(files));
            } catch (err) {
              logToFile(`[ERROR] List Images Failed: ${err.message}`);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: 'Failed to list files' }));
            }
            return;
          }

          // Serve old_pic files
          if (req.url?.startsWith('/old_pic/')) {
            const fileName = decodeURIComponent(req.url.replace('/old_pic/', ''));
            const filePath = path.join(__dirname, 'old_pic', fileName);
            
            if (fs.existsSync(filePath)) {
               const stat = fs.statSync(filePath);
               res.setHeader('Content-Length', stat.size);
               res.setHeader('Content-Type', 'image/png'); // Simplified content type
               const readStream = fs.createReadStream(filePath);
               readStream.pipe(res);
               return;
            }
          }

          next();
        });
      }
    });

    return {
      server: {
        port: 3400,
        host: '0.0.0.0',
        proxy: {
          '/gallery': {
            target: 'http://100.76.199.94:3002',
            changeOrigin: true,
            secure: false,
            rewrite: (path) => path.replace(/^\/gallery/, '')
          }
        }
      },
      plugins: [react(), fileSystemPlugin()],
      define: {
        'process.env.API_KEY': JSON.stringify(env.GEMINI_API_KEY),
        'process.env.GEMINI_API_KEY': JSON.stringify(env.GEMINI_API_KEY)
      },
      resolve: {
        alias: {
          '@': path.resolve(__dirname, '.'),
        }
      }
    };
});
