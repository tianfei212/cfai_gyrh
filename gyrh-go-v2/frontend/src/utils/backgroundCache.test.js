import assert from 'node:assert/strict';
import test from 'node:test';
import { createBackgroundCache } from './backgroundCache.js';

test('background cache reuses a loaded page without refetching', async () => {
  const requests = [];
  const cache = createBackgroundCache({
    fetchPage: async ({ page, limit }) => {
      requests.push({ page, limit });
      return {
        items: [{ id: page, name: `page ${page}` }],
        total: 12,
      };
    },
  });

  const first = await cache.loadPage(1, { limit: 6 });
  const second = await cache.loadPage(1, { limit: 6 });

  assert.deepEqual(first.items, [{ id: 1, name: 'page 1' }]);
  assert.strictEqual(second, first);
  assert.deepEqual(requests, [{ page: 1, limit: 6 }]);
});

test('background cache can invalidate one page or all pages', async () => {
  let version = 0;
  const cache = createBackgroundCache({
    fetchPage: async ({ page }) => ({
      items: [{ id: page, version: ++version }],
      total: 1,
    }),
  });

  const pageOne = await cache.loadPage(1);
  const pageTwo = await cache.loadPage(2);
  cache.invalidatePage(1);

  const refreshedPageOne = await cache.loadPage(1);
  const cachedPageTwo = await cache.loadPage(2);
  cache.invalidateAll();
  const refreshedPageTwo = await cache.loadPage(2);

  assert.notStrictEqual(refreshedPageOne, pageOne);
  assert.strictEqual(cachedPageTwo, pageTwo);
  assert.notStrictEqual(refreshedPageTwo, pageTwo);
});
