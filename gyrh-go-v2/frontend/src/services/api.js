export async function signRequest(clientIP, publicKey, privateKey) {
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

export async function getAuthHeaders() {
  const clientIP = '127.0.0.1';
  const publicKey = import.meta.env.VITE_GYRH_AUTH_PUBLIC_KEY || 'gyrh_web';
  const privateKey = import.meta.env.VITE_GYRH_AUTH_PRIVATE_KEY || 'secret';

  return signRequest(clientIP, publicKey, privateKey);
}

export function buildFrontendAuthHeader(session) {
  if (!session?.token) return {};
  return {
    Authorization: `Bearer ${session.token}`,
  };
}

export async function fetchApi(url, options = {}) {
  // In a real app, you might want to get the client IP or let a gateway handle it
  const authHeaders = await getAuthHeaders();
  const frontendAuthHeaders = buildFrontendAuthHeader(getStoredFrontendSessionSafe());

  const headers = {
    'Content-Type': 'application/json',
    ...authHeaders,
    ...frontendAuthHeaders,
    ...options.headers,
  };

  const method = options.method || 'GET';
  let logBody = options.body;
  if (typeof options.body === 'string') {
    try {
      logBody = JSON.parse(options.body);
    } catch (e) {
      // ignore
    }
  }
  console.log(`[API Request] ${method} ${url}`, logBody || '');

  try {
    const response = await fetch(url, {
      ...options,
      headers,
      credentials: 'include',
    });

    const responseText = await response.text();
    let data;
    try {
      data = JSON.parse(responseText);
    } catch (e) {
      console.error(`[API Parse Error] ${method} ${url} - Response: ${responseText}`);
      throw new Error('Invalid JSON response from server');
    }

    if (data.code !== 0) {
      console.error(`[API Error] ${method} ${url} - Code: ${data.code}, Message: ${data.message}`);
      throw new Error(data.message || 'API Request Failed');
    }

    console.log(`[API Success] ${method} ${url}`, data.data);
    return data.data;
  } catch (error) {
    console.error(`[API Failed] ${method} ${url}`, error);
    throw error;
  }
}

function getStoredFrontendSessionSafe() {
  try {
    const raw = globalThis.localStorage?.getItem('gyrh_frontend_session');
    return raw ? JSON.parse(raw) : null;
  } catch (err) {
    return null;
  }
}
