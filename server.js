const express = require('express');
const path = require('path');
const fs = require('fs');
const dotenv = require('dotenv');
const cors = require('cors');
const { createProxyMiddleware } = require('http-proxy-middleware');

// Load environment variables
dotenv.config({ path: '.env.local' });
dotenv.config(); // Fallback to .env

const app = express();
const PORT = process.env.PORT || 3400;

// Middleware
app.use(cors());
// Increase limit for image uploads
app.use(express.json({ limit: '50mb' }));
app.use(express.urlencoded({ limit: '50mb', extended: true }));

// Setup Logger
const logsDir = path.join(__dirname, 'logs');
if (!fs.existsSync(logsDir)) {
  fs.mkdirSync(logsDir, { recursive: true });
}

const getLogFilePath = () => {
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, '0');
  const day = String(now.getDate()).padStart(2, '0');
  const timestampStr = `${year}${month}${day}`;
  return path.join(logsDir, `${timestampStr}.log`);
};

const logToFile = (message) => {
  const time = new Date().toISOString();
  const logLine = `[${time}] ${message}\n`;
  try {
    fs.appendFileSync(getLogFilePath(), logLine);
    process.stdout.write(logLine);
  } catch (e) {
    console.error("Failed to write log:", e);
  }
};

const logToServer = (...args) => {
  const msg = args
    .map((item) => {
      if (typeof item === 'string') return item;
      try {
        return JSON.stringify(item);
      } catch (_) {
        return String(item);
      }
    })
    .join(' ');
  logToFile(msg);
};

// Helper Functions
const getKey = (req, service = 'google') => {
  // 1. Try to get key from request header (custom header)
  const headerKey = req.headers['x-api-key'];
  if (headerKey) return headerKey;
  
  // 2. Fallback to server-side env based on service
  if (service === 'aliwan') {
      return process.env.ALI_API_KEY || '';
  }
  return process.env.GEMINI_API_KEY || process.env.API_KEY || '';
};

// ... inside handlers, pass req to getKey ...
// Example: const apiKey = getKey(req);

const buildPrompt = (type, userPrompt) => {
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
  } catch (e) {
    logToFile(`[ERROR] Failed to load prompts: ${e.message}`);
  }
  return '';
};

const postGoogle = async (req, model, parts, generationConfig, systemText) => {
  const apiKey = getKey(req);
  if (!apiKey) throw new Error('API Key missing');
  
  const url = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${apiKey}`;
  const start = Date.now();
  const safetySettings = [
    { category: 'HARM_CATEGORY_HARASSMENT', threshold: 'BLOCK_NONE' },
    { category: 'HARM_CATEGORY_HATE_SPEECH', threshold: 'BLOCK_NONE' },
    { category: 'HARM_CATEGORY_SEXUALLY_EXPLICIT', threshold: 'BLOCK_NONE' },
    { category: 'HARM_CATEGORY_DANGEROUS_CONTENT', threshold: 'BLOCK_NONE' }
  ];
  
  try {
    // Dynamic import for fetch in older Node versions if needed, or assume Node 18+ global fetch
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
    
    if (!resp.ok || json.error) {
      logToFile(`[ERROR] Google API Failed (${resp.status}): ${JSON.stringify(json)}`);
    } else if (!json.candidates?.[0]?.content?.parts?.[0]?.inline_data && !json.candidates?.[0]?.content?.parts?.[0]?.text) {
      logToFile(`[WARN] No inline_data/text in response (Safety?): ${JSON.stringify(json)}`);
    }

    logToFile(`[INFO] Google Call ${model} - ${Date.now() - start}ms`);
    return json;
  } catch (error) {
    logToFile(`[ERROR] Google API Network Error: ${error.message}`);
    throw error;
  }
};

const extractImageBase64 = (json) => {
  const parts = json?.candidates?.[0]?.content?.parts;
  if (Array.isArray(parts)) {
    for (const part of parts) {
      const data = part?.inline_data?.data || part?.inlineData?.data;
      if (data) return 'data:image/png;base64,' + data;
    }
  }
  return '';
};

// --- API Routes ---

// Basic request logging (API only)
app.use('/api', (req, res, next) => {
  const start = Date.now();
  const requestMeta = {
    method: req.method,
    path: req.originalUrl,
    ip: req.ip,
    ua: req.headers['user-agent'] || ''
  };
  logToFile(`[REQ] ${JSON.stringify(requestMeta)}`);
  res.on('finish', () => {
    logToFile(`[RES] ${req.method} ${req.originalUrl} -> ${res.statusCode} (${Date.now() - start}ms)`);
  });
  next();
});

// Health check
app.get(['/api/health', '/healthz'], (req, res) => {
  res.json({
    ok: true,
    service: 'gyrh-server',
    uptimeSec: Number(process.uptime().toFixed(2)),
    timestamp: new Date().toISOString()
  });
});

// Logger API
app.post('/api/log', (req, res) => {
  try {
    const { message, details, level = 'INFO' } = req.body;
    let logMsg = `[${level}] ${message}`;
    if (details) {
      logMsg += ` - ${JSON.stringify(details)}`;
    }
    logToFile(logMsg);
    res.json({ success: true });
  } catch (err) {
    console.error('Error processing log:', err);
    res.status(500).json({ error: 'Failed to process log' });
  }
});

// Save Image API
app.post('/api/save-image', (req, res) => {
  try {
    const { name, data } = req.body;
    
    if (!name || !data) {
      logToFile(`[ERROR] Save Image Failed: Missing name or data`);
      return res.status(400).json({ error: 'Missing name or data' });
    }

    const dir = path.join(__dirname, 'old_pic');
    if (!fs.existsSync(dir)) {
      fs.mkdirSync(dir, { recursive: true });
    }

    const base64Data = data.replace(/^data:image\/\w+;base64,/, "");
    const buffer = Buffer.from(base64Data, 'base64');
    
    const filePath = path.join(dir, name);
    fs.writeFileSync(filePath, buffer);
    
    logToFile(`[INFO] Image Saved: ${name}`);
    res.json({ success: true, path: filePath });
  } catch (err) {
    logToFile(`[ERROR] Save Image Failed: ${err.message}`);
    res.status(500).json({ error: 'Failed to save file' });
  }
});

// Proxy API for external requests (like AliWan)
const handleAliwanProxy = async (req, res) => {
  logToServer(`[DEBUG] AliWan proxy hit: ${req.method} ${req.originalUrl}`);
  const targetUrl = req.headers['x-target-url'];
  const apiKey = req.headers['x-api-key'] || process.env.ALI_API_KEY;

  if (!targetUrl || !apiKey) {
    logToServer(`[ERROR] Proxy AliWan missing headers. targetUrl: ${!!targetUrl}, apiKey: ${!!apiKey}`);
    return res.status(400).json({ error: 'Missing target url or api key' });
  }

  logToServer(`[DEBUG] Proxying AliWan to: ${targetUrl}`);

  try {
    const headers = {
      'Authorization': `Bearer ${apiKey}`,
      'Content-Type': 'application/json'
    };
    
    // Forward X-DashScope-Async header if present
    if (req.headers['x-dashscope-async']) {
      headers['X-DashScope-Async'] = req.headers['x-dashscope-async'];
    }

    const fetchOptions = {
      method: req.method,
      headers: headers
    };
    
    // Only attach body for POST/PUT requests
    if (req.method !== 'GET' && req.method !== 'HEAD') {
        // Ensure we don't send an empty string or empty object if there's no actual body
        if (req.body && Object.keys(req.body).length > 0) {
            fetchOptions.body = JSON.stringify(req.body);
            logToServer(`[DEBUG] Proxy AliWan attached body size: ${fetchOptions.body.length}`);
        }
    }

    const upstreamStart = Date.now();
    const response = await fetch(targetUrl, fetchOptions);

    logToServer(`[DEBUG] Proxy AliWan response status from Alibaba: ${response.status} ${response.statusText}, latency=${Date.now() - upstreamStart}ms`);
    const data = await response.text();
    const contentType = response.headers.get('content-type') || 'application/json; charset=utf-8';
    res.setHeader('Content-Type', contentType);
    res.status(response.status).send(data);
  } catch (err) {
    logToFile(`[ERROR] Proxy AliWan Failed (Network Level): ${err.message}\n${err.stack || ''}`);
    res.status(500).json({ error: 'Proxy failed', details: err.message, route: req.originalUrl });
  }
};

// Keep backward-compatible route aliases for clients
[
  '/api/proxy-aliwan',
  '/api/proxy_aliwan',
  '/api/proxyAliwan',
  '/api/proxy-wan'
].forEach((routePath) => app.all(routePath, handleAliwanProxy));

// Proxy API for downloading external images (bypassing CORS)
app.get('/api/proxy-image', async (req, res) => {
  const targetUrl = req.query.url;

  if (!targetUrl) {
    return res.status(400).json({ error: 'Missing url parameter' });
  }

  try {
    logToFile(`[INFO] Proxying image download: ${targetUrl}`);
    
    // Fetch directly from the target URL
    const response = await fetch(targetUrl);
    
    if (!response.ok) {
      logToFile(`[ERROR] Proxy fetch failed: ${response.status} ${response.statusText}`);
      return res.status(response.status).send(`Proxy fetch failed: ${response.statusText}`);
    }

    // Pass necessary headers
    const contentType = response.headers.get('content-type');
    if (contentType) res.setHeader('Content-Type', contentType);
    const contentLength = response.headers.get('content-length');
    if (contentLength) res.setHeader('Content-Length', contentLength);
    
    res.setHeader('Access-Control-Allow-Origin', '*');

    const arrayBuffer = await response.arrayBuffer();
    res.send(Buffer.from(arrayBuffer));
    logToFile(`[INFO] Proxy download success: ${targetUrl}`);
  } catch (err) {
    logToFile(`[ERROR] Proxy Image Failed: ${err.message}`);
    res.status(500).json({ error: 'Proxy failed', details: err.message });
  }
});

// Compose API
app.post('/api/compose', async (req, res) => {
  try {
    const { bgBase64, selfieBase64 } = req.body;
    if (!getKey(req)) return res.status(401).json({ error: 'INVALID_KEY' });
    if (!bgBase64 || !selfieBase64) return res.status(400).json({ error: 'BAD_REQUEST' });

    const parts = [
      { inline_data: { mime_type: 'image/jpeg', data: bgBase64.replace(/^data:image\/\w+;base64,/, '') } },
      { inline_data: { mime_type: 'image/jpeg', data: selfieBase64.replace(/^data:image\/\w+;base64,/, '') } }
    ];
    const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
    const json = await postGoogle(req, 'gemini-3-pro-image-preview', parts, genCfg, buildPrompt('composite'));
    res.json({ base64: extractImageBase64(json) });
  } catch (err) {
    logToFile(`[ERROR] Compose Failed: ${err.message}`);
    res.status(500).json({ error: 'NET_ERROR' });
  }
});

// Edit API
app.post('/api/edit', async (req, res) => {
  try {
    const { imageBase64, prompt } = req.body;
    if (!getKey(req)) return res.status(401).json({ error: 'INVALID_KEY' });
    if (!imageBase64 || typeof prompt !== 'string') return res.status(400).json({ error: 'BAD_REQUEST' });

    const parts = [
      { inline_data: { mime_type: 'image/png', data: imageBase64.replace(/^data:image\/\w+;base64,/, '') } }
    ];
    const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
    const json = await postGoogle(req, 'gemini-3-pro-image-preview', parts, genCfg, buildPrompt('edit', prompt));
    res.json({ base64: extractImageBase64(json) });
  } catch (err) {
    logToFile(`[ERROR] Edit Failed: ${err.message}`);
    res.status(500).json({ error: 'NET_ERROR' });
  }
});

// Upscale API
app.post('/api/upscale', async (req, res) => {
  try {
    const { imageBase64 } = req.body;
    if (!getKey(req)) return res.status(401).json({ error: 'INVALID_KEY' });
    if (!imageBase64) return res.status(400).json({ error: 'BAD_REQUEST' });

    const parts = [
      { inline_data: { mime_type: 'image/png', data: imageBase64.replace(/^data:image\/\w+;base64,/, '') } }
    ];
    const genCfg = { temperature: 0.2, top_p: 0.3, seed: 42 };
    const json = await postGoogle(req, 'gemini-3-pro-image-preview', parts, genCfg, buildPrompt('upscale'));
    res.json({ base64: extractImageBase64(json) });
  } catch (err) {
    logToFile(`[ERROR] Upscale Failed: ${err.message}`);
    res.status(500).json({ error: 'NET_ERROR' });
  }
});

// Transcribe API
app.post('/api/transcribe', async (req, res) => {
  try {
    const { audioBase64 } = req.body;
    if (!getKey(req)) return res.status(401).json({ error: 'INVALID_KEY' });
    if (!audioBase64) return res.status(400).json({ error: 'BAD_REQUEST' });

    const p = JSON.parse(fs.readFileSync(path.join(__dirname, 'prompts.json'), 'utf-8'));
    const parts = [
      { text: p.transcribe.prompt.en },
      { inline_data: { mime_type: 'audio/webm', data: audioBase64.replace(/^data:audio\/\w+;base64,/, '') } }
    ];
    const json = await postGoogle(req, 'gemini-2.5-flash', parts, { temperature: 0, top_p: 0.3 });
    let text = '';
    const respParts = json?.candidates?.[0]?.content?.parts;
    if (Array.isArray(respParts)) {
      for (const part of respParts) {
        if (part?.text) { text = part.text; break; }
      }
    }
    res.json({ text: text?.trim() || '' });
  } catch (err) {
    logToFile(`[ERROR] Transcribe Failed: ${err.message}`);
    res.status(500).json({ error: 'NET_ERROR' });
  }
});

// List Images API
app.get('/api/list-images', (req, res) => {
  try {
    const dir = path.join(__dirname, 'old_pic');
    if (!fs.existsSync(dir)) {
      return res.json([]);
    }

    const files = fs.readdirSync(dir)
      .filter(file => /\.(png|jpg|jpeg|webp)$/i.test(file))
      .map(file => {
        const stats = fs.statSync(path.join(dir, file));
        return {
          name: file,
          timestamp: stats.mtimeMs,
          url: `/old_pic/${file}`
        };
      })
      .sort((a, b) => b.timestamp - a.timestamp);

    logToFile(`[INFO] List Images: Found ${files.length} files`);
    res.json(files);
  } catch (err) {
    logToFile(`[ERROR] List Images Failed: ${err.message}`);
    res.status(500).json({ error: 'Failed to list files' });
  }
});

// Delete Images API
app.post('/api/delete-images', (req, res) => {
  try {
    const { names } = req.body;
    
    if (!names || !Array.isArray(names)) {
      return res.status(400).json({ error: 'Missing names array' });
    }

    const dir = path.join(__dirname, 'old_pic');
    let deletedCount = 0;
    const errors = [];

    names.forEach(name => {
      const filePath = path.join(dir, name);
      try {
        if (fs.existsSync(filePath)) {
          fs.unlinkSync(filePath);
          deletedCount++;
        }
      } catch (e) {
        errors.push({ name, error: e.message });
      }
    });

    logToFile(`[INFO] Deleted ${deletedCount} images. Errors: ${errors.length}`);
    res.json({ success: true, deletedCount, errors });
  } catch (err) {
    logToFile(`[ERROR] Delete Images Failed: ${err.message}`);
    res.status(500).json({ error: 'Failed to delete files' });
  }
});

// Proxy Gallery API
app.use('/gallery', createProxyMiddleware({
  target: 'http://100.76.199.94:3002',
  changeOrigin: true,
  pathRewrite: { '^/gallery': '' },
}));

// Serve Static Files (old_pic)
app.use('/old_pic', express.static(path.join(__dirname, 'old_pic')));

// Serve Frontend (dist)
app.use(express.static(path.join(__dirname, 'dist')));

// SPA Fallback
app.use((req, res, next) => {
  if (req.method === 'GET' && !req.path.startsWith('/api')) {
    res.sendFile(path.join(__dirname, 'dist', 'index.html'));
  } else {
    next();
  }
});

// Start Server
app.listen(PORT, () => {
  logToFile(`Server running on port ${PORT}`);
  logToFile(`Environment: ${process.env.NODE_ENV || 'development'}`);
  console.log(`Server started at http://localhost:${PORT}`);
});
