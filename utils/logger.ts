
export const logToServer = async (message: string, details?: any, level: 'INFO' | 'ERROR' | 'WARN' = 'INFO') => {
  // Console log in browser for debugging
  console.log(`[${level}] ${message}`, details || '');

  try {
    await fetch('/api/log', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        message,
        details,
        level
      }),
    });
  } catch (e) {
    // Fail silently to avoid recursion or blocking UI
    console.error("Failed to send log to server", e);
  }
};
