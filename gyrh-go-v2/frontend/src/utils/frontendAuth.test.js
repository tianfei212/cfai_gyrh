import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  buildLoginRedirectPath,
  buildFrontendLoginHeaders,
  canAccessPath,
  getPostLoginPath,
  getStoredFrontendSession,
  logoutFrontend,
  setStoredFrontendSession,
} from '../services/frontendAuth.js';

test('builds login headers with HOME1 and HOME2', () => {
  assert.deepEqual(buildFrontendLoginHeaders('admin', '123456'), {
    HOME1: 'admin',
    HOME2: '123456',
  });
});

test('stores and restores frontend session token', () => {
  const values = new Map();
  const storage = {
    getItem: (key) => values.get(key) || null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  };

  setStoredFrontendSession({ username: 'admin', role: 'admin', token: 'jwt-token' }, storage);

  assert.deepEqual(getStoredFrontendSession(storage), {
    username: 'admin',
    role: 'admin',
    token: 'jwt-token',
  });
});

test('does not persist frontend password in session storage', () => {
  const values = new Map();
  const storage = {
    getItem: (key) => values.get(key) || null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  };

  setStoredFrontendSession({
    username: 'admin',
    role: 'admin',
    token: 'jwt-token',
    password: 'secret@#',
  }, storage);

  const raw = values.get('gyrh_frontend_session');
  assert.equal(raw.includes('secret@#'), false);
  assert.equal(raw.includes('password'), false);
});

test('logout clears stored frontend session', async () => {
  const values = new Map();
  const storage = {
    getItem: (key) => values.get(key) || null,
    setItem: (key, value) => values.set(key, value),
    removeItem: (key) => values.delete(key),
  };
  const originalFetch = globalThis.fetch;
  globalThis.localStorage = storage;
  globalThis.fetch = async () => ({ ok: true, json: async () => ({ code: 0 }) });

  try {
    setStoredFrontendSession({ username: 'admin', role: 'admin', token: 'jwt-token' }, storage);
    await logoutFrontend({ token: 'jwt-token' });
    assert.equal(getStoredFrontendSession(storage), null);
  } finally {
    globalThis.fetch = originalFetch;
    delete globalThis.localStorage;
  }
});

test('checks path access by role', () => {
  assert.equal(canAccessPath('/admin_viewer', { role: 'admin' }), true);
  assert.equal(canAccessPath('/', { role: 'admin' }), true);
  assert.equal(canAccessPath('/admin_viewer', { role: 'pshow' }), false);
  assert.equal(canAccessPath('/', { role: 'pshow' }), true);
  assert.equal(canAccessPath('/', null), false);
});

test('builds login redirect path with original target', () => {
  assert.equal(buildLoginRedirectPath('/', ''), '/login?next=%2F');
  assert.equal(buildLoginRedirectPath('/admin_viewer', '?tab=skills'), '/login?next=%2Fadmin_viewer%3Ftab%3Dskills');
  assert.equal(buildLoginRedirectPath('/login', ''), '/login');
});

test('chooses post-login path by role', () => {
  assert.equal(getPostLoginPath({ role: 'admin' }, '/'), '/admin_viewer');
  assert.equal(getPostLoginPath({ role: 'admin' }, '/foo'), '/admin_viewer');
  assert.equal(getPostLoginPath({ role: 'pshow' }, '/'), '/');
  assert.equal(getPostLoginPath({ role: 'pshow' }, '/admin_viewer'), '/');
  assert.equal(getPostLoginPath({ role: 'pshow' }, '/foo'), '/foo');
});
