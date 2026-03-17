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
          const readBody = async () => {
            return new Promise<any>((resolve, reject) => {
              const chunks: Buffer[] = [];
              req.on('data', c => chunks.push(c));
              req.on('end', () => {
                try {
                  const body = JSON.parse(Buffer.concat(chunks).toString() || '{}');
                  resolve(body);
                } catch (e) {
                  reject(e);
                }
              });
              req.on('error', reject);
            });
          };
          const getKey = () => {
            const k = process.env.GEMINI_API_KEY || process.env.API_KEY || env.GEMINI_API_KEY;
            return k || '';
          };
          const buildPrompt = (type: 'composite' | 'edit' | 'upscale', userPrompt?: string) => {
            try {
              const p = JSON.parse(fs.readFileSync(path.join(__dirname, 'prompts.json'), 'utf-8'));
              if (type === 'composite') {
                return [
                  'ROLE: ' + p.composite.role.en,
                  'TASK: ' + p.composite.task.en,
                  'ANATOMY & IDENTITY RULES (CRITICAL):',
                  p.composite.rules.anatomy.en,
                  'COMPOSITION & LIGHTING:',
                  p.composite.rules.composition.en,
                  'OUTPUT: ' + p.composite.output.en
                ].join('\n');
              }
              if (type === 'edit') {
                const t = String(p.edit.task.en).replace('{userPrompt}', userPrompt || '');
                return ['ROLE: ' + p.edit.role.en, 'TASK: ' + t, 'STRICT CONSTRAINTS:', p.edit.constraints.en].join('\n');
              }
              if (type === 'upscale') {
                return ['ROLE: ' + p.upscale.role.en, 'TASK: ' + p.upscale.task.en, 'INSTRUCTIONS:', p.upscale.instructions.en].join('\n');
              }
            } catch {}
            return '';
          };
          const postGoogle = async (model: string, parts: any[], generationConfig?: any, systemText?: string) => {
            const apiKey = getKey();
            const url = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${apiKey}`;
            const start = Date.now();
            const safetySettings = [
              { category: 'HARM_CATEGORY_HARASSMENT', threshold: 'BLOCK_NONE' },
              { category: 'HARM_CATEGORY_HATE_SPEECH', threshold: 'BLOCK_NONE' },
              { category: 'HARM_CATEGORY_SEXUALLY_EXPLICIT', threshold: 'BLOCK_NONE' },
              { category: 'HARM_CATEGORY_DANGEROUS_CONTENT', threshold: 'BLOCK_NONE' }
            ];
            const resp = await fetch(url, {
              method: 'POST',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({
                contents: [{ role: 'user', parts }],
                system_instruction: systemText ? { role: 'system', parts: [{ text: systemText }] } : undefined,
                generationConfig,
                safetySettings
              })
            });
            const json = await resp.json();
            
            // Error Diagnostic Logging
            if (!resp.ok || json.error) {
              logToFile(`[ERROR] Google API Failed (${resp.status}): ${JSON.stringify(json)}`);
            } else if (!json.candidates?.[0]?.content?.parts?.[0]?.inline_data && !json.candidates?.[0]?.content?.parts?.[0]?.text) {
              logToFile(`[WARN] No inline_data/text in response (Safety?): ${JSON.stringify(json)}`);
            }

            logToFile(`[INFO] Google Call ${model} - ${Date.now() - start}ms`);
            return json;
          };
          const extractImageBase64 = (json: any) => {
            const parts = json?.candidates?.[0]?.content?.parts;
            if (Array.isArray(parts)) {
              for (const part of parts) {
                // Compatible with both inline_data (REST) and inlineData (SDK-style/Protobuf)
                const data = part?.inline_data?.data || part?.inlineData?.data;
                if (data) return 'data:image/png;base64,' + data;
              }
            }
            return '';
          };
          const composeCall = async (bgBase64: string, selfieBase64: string) => {
            const parts = [
              { inline_data: { mime_type: 'image/jpeg', data: bgBase64.replace(/^data:image\/\w+;base64,/, '') } },
              { inline_data: { mime_type: 'image/jpeg', data: selfieBase64.replace(/^data:image\/\w+;base64,/, '') } }
            ];
            const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
            const json = await postGoogle('gemini-3-pro-image-preview', parts, genCfg, buildPrompt('composite'));
            return { base64: extractImageBase64(json) };
          };
          const editCall = async (imageBase64: string, promptText: string) => {
            const parts = [
              { inline_data: { mime_type: 'image/png', data: imageBase64.replace(/^data:image\/\w+;base64,/, '') } }
            ];
            const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
            const json = await postGoogle('gemini-3-pro-image-preview', parts, genCfg, buildPrompt('edit', promptText));
            return { base64: extractImageBase64(json) };
          };
          const upscaleCall = async (imageBase64: string) => {
            const parts = [
              { inline_data: { mime_type: 'image/png', data: imageBase64.replace(/^data:image\/\w+;base64,/, '') } }
            ];
            const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
            const json = await postGoogle('gemini-3-pro-image-preview', parts, genCfg, buildPrompt('upscale'));
            return { base64: extractImageBase64(json) };
          };
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

          if (req.url === '/api/compose' && req.method === 'POST') {
            try {
              const body = await readBody();
              const { bgBase64, selfieBase64 } = body || {};
              if (!getKey()) {
                res.statusCode = 401;
                res.end(JSON.stringify({ error: 'INVALID_KEY' }));
                return;
              }
              if (!bgBase64 || !selfieBase64) {
                res.statusCode = 400;
                res.end(JSON.stringify({ error: 'BAD_REQUEST' }));
                return;
              }
              const out = await composeCall(bgBase64, selfieBase64);
              res.setHeader('Content-Type', 'application/json');
              res.end(JSON.stringify({ base64: out.base64 }));
            } catch (err: any) {
              logToFile(`[ERROR] Compose Failed: ${err.message}`);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: 'NET_ERROR' }));
            }
            return;
          }

          if (req.url === '/api/edit' && req.method === 'POST') {
            try {
              const body = await readBody();
              const { imageBase64, prompt } = body || {};
              if (!getKey()) {
                res.statusCode = 401;
                res.end(JSON.stringify({ error: 'INVALID_KEY' }));
                return;
              }
              if (!imageBase64 || typeof prompt !== 'string') {
                res.statusCode = 400;
                res.end(JSON.stringify({ error: 'BAD_REQUEST' }));
                return;
              }
              const out = await editCall(imageBase64, prompt);
              res.setHeader('Content-Type', 'application/json');
              res.end(JSON.stringify({ base64: out.base64 }));
            } catch (err: any) {
              logToFile(`[ERROR] Edit Failed: ${err.message}`);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: 'NET_ERROR' }));
            }
            return;
          }

          if (req.url === '/api/upscale' && req.method === 'POST') {
            try {
              const body = await readBody();
              const { imageBase64 } = body || {};
              if (!getKey()) {
                res.statusCode = 401;
                res.end(JSON.stringify({ error: 'INVALID_KEY' }));
                return;
              }
              if (!imageBase64) {
                res.statusCode = 400;
                res.end(JSON.stringify({ error: 'BAD_REQUEST' }));
                return;
              }
              const out = await upscaleCall(imageBase64);
              res.setHeader('Content-Type', 'application/json');
              res.end(JSON.stringify({ base64: out.base64 }));
            } catch (err: any) {
              logToFile(`[ERROR] Upscale Failed: ${err.message}`);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: 'NET_ERROR' }));
            }
            return;
          }

          if (req.url === '/api/transcribe' && req.method === 'POST') {
            try {
              const body = await readBody();
              const { audioBase64 } = body || {};
              if (!getKey()) {
                res.statusCode = 401;
                res.end(JSON.stringify({ error: 'INVALID_KEY' }));
                return;
              }
              if (!audioBase64) {
                res.statusCode = 400;
                res.end(JSON.stringify({ error: 'BAD_REQUEST' }));
                return;
              }
              const p = JSON.parse(fs.readFileSync(path.join(__dirname, 'prompts.json'), 'utf-8'));
              const parts = [
                { text: p.transcribe.prompt.en },
                { inline_data: { mime_type: 'audio/webm', data: audioBase64.replace(/^data:audio\/\w+;base64,/, '') } }
              ];
              const json = await postGoogle('gemini-2.5-flash', parts, { temperature: 0, top_p: 0.3 });
              let text = '';
              const respParts = json?.candidates?.[0]?.content?.parts;
              if (Array.isArray(respParts)) {
                for (const part of respParts) {
                  if (part?.text) { text = part.text; break; }
                }
              }
              res.setHeader('Content-Type', 'application/json');
              res.end(JSON.stringify({ text: text?.trim() || '' }));
            } catch (err: any) {
              logToFile(`[ERROR] Transcribe Failed: ${err.message}`);
              res.statusCode = 500;
              res.end(JSON.stringify({ error: 'NET_ERROR' }));
            }
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
        allowedHosts: ['mqia.chinafilmai.com'],
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
