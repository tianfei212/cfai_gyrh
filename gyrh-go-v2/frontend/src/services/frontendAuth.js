const FRONTEND_SESSION_KEY = 'gyrh_frontend_session';

export function buildFrontendLoginHeaders(username, password) {
  return {
    HOME1: username,
    HOME2: password,
  };
}

export function getStoredFrontendSession(storage = globalThis.localStorage) {
  if (!storage) return null;
  const raw = storage.getItem(FRONTEND_SESSION_KEY);
  if (!raw) return null;
  try {
    const session = JSON.parse(raw);
    return session && session.token ? session : null;
  } catch (err) {
    storage.removeItem(FRONTEND_SESSION_KEY);
    return null;
  }
}

export function setStoredFrontendSession(session, storage = globalThis.localStorage) {
  if (!storage) return;
  if (!session || !session.token) {
    storage.removeItem(FRONTEND_SESSION_KEY);
    return;
  }
  storage.setItem(FRONTEND_SESSION_KEY, JSON.stringify({
    username: session.username,
    role: session.role,
    token: session.token,
  }));
}

export function clearStoredFrontendSession(storage = globalThis.localStorage) {
  if (storage) storage.removeItem(FRONTEND_SESSION_KEY);
}

export function canAccessPath(path, session) {
  if (!session || !session.role) return false;
  if (path.startsWith('/login')) return true;
  if (path.startsWith('/admin_viewer')) return session.role === 'admin';
  return session.role === 'admin' || session.role === 'pshow';
}

export function buildLoginRedirectPath(path = '/', search = '') {
  const target = `${path || '/'}${search || ''}`;
  if (path.startsWith('/login')) return '/login';
  return `/login?next=${encodeURIComponent(target)}`;
}

export function getPostLoginPath(session, nextPath = '/') {
  if (session?.role === 'admin') return '/admin_viewer';
  if (session?.role === 'pshow' && nextPath && !nextPath.startsWith('/admin_viewer') && !nextPath.startsWith('/login')) {
    return nextPath;
  }
  return '/';
}

export async function loginFrontend(username, password) {
  const response = await fetch('/api/v1/frontend-auth/login', {
    method: 'POST',
    headers: buildFrontendLoginHeaders(username, password),
    credentials: 'include',
  });
  const data = await response.json();
  if (!response.ok || data.code !== 0) {
    throw new Error(data.message || '登录失败');
  }
  setStoredFrontendSession(data.data);
  return data.data;
}

export async function logoutFrontend(session = getStoredFrontendSession()) {
  try {
    await fetch('/api/v1/frontend-auth/logout', {
      method: 'POST',
      headers: session?.token ? { Authorization: `Bearer ${session.token}` } : {},
      credentials: 'include',
    });
  } finally {
    clearStoredFrontendSession();
  }
}

export async function validateFrontendSession(session = getStoredFrontendSession()) {
  if (!session?.token) return null;
  const response = await fetch('/api/v1/frontend-auth/session', {
    headers: {
      Authorization: `Bearer ${session.token}`,
    },
    credentials: 'include',
  });
  if (!response.ok) {
    clearStoredFrontendSession();
    return null;
  }
  const data = await response.json();
  if (data.code !== 0) {
    clearStoredFrontendSession();
    return null;
  }
  return { ...session, ...data.data, token: session.token };
}
