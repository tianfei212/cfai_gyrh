const crypto = require('crypto');

async function test() {
  const url = 'http://127.0.0.1:9913/api/v1/background-prompts/sync-english';
  const clientIP = '127.0.0.1';
  const publicKey = '456973f56716801b00a61339bdc307f3153162d5247ca4e883b09dc097cccb10';
  const privateKey = 'e2a60d4061a429294746f311218be43c505e663d036b0528a90636cfa04258b5';
  
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const content = clientIP + publicKey + timestamp;

  const key = crypto.createHmac('sha256', privateKey);
  key.update(content);
  const signature = key.digest('hex');

  const headers = {
    'Content-Type': 'application/json',
    'X-Real-IP': clientIP,
    'X-Public-Key': publicKey,
    'X-Timestamp': timestamp,
    'X-Signature': signature,
  };

  const body = JSON.stringify({
    wan_prompt_zh: "测试",
    wan_negative_prompt_zh: "",
    gemini_prompt_zh: "",
    gemini_negative_prompt_zh: ""
  });

  try {
    console.log("Sending request...");
    const response = await fetch(url, { method: 'POST', headers, body });
    const text = await response.text();
    console.log("Response:", response.status, text);
  } catch (err) {
    console.error("Error:", err);
  }
}

test();
