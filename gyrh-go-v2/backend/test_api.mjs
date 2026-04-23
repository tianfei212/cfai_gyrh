import crypto from 'crypto';

async function signRequest(clientIP, publicKey, privateKey) {
  const timestamp = Math.floor(Date.now() / 1000).toString();
  const content = clientIP + publicKey + timestamp;

  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(privateKey),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"]
  );

  const signatureBuffer = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(content)
  );

  const signature = Array.from(new Uint8Array(signatureBuffer))
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  return {
    "X-Real-IP": clientIP,
    "X-Public-Key": publicKey,
    "X-Timestamp": timestamp,
    "X-Signature": signature,
  };
}

async function run() {
  const headers = await signRequest('127.0.0.1', 'gyrh_web', 'secret');
  const res = await fetch('http://127.0.0.1:9913/api/v1/images?limit=1', { headers });
  const data = await res.json();
  console.log(JSON.stringify(data, null, 2));
}
run();
