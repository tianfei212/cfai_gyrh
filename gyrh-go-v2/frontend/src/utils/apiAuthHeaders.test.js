import assert from 'node:assert/strict';
import { test } from 'node:test';
import { buildFrontendAuthHeader } from '../services/api.js';

test('builds authorization header from frontend token', () => {
  assert.deepEqual(buildFrontendAuthHeader({ token: 'jwt-token' }), {
    Authorization: 'Bearer jwt-token',
  });
});

test('omits authorization header without token', () => {
  assert.deepEqual(buildFrontendAuthHeader(null), {});
  assert.deepEqual(buildFrontendAuthHeader({ token: '' }), {});
});
