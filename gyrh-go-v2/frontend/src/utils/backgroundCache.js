export function createBackgroundCache({ fetchPage } = {}) {
  if (typeof fetchPage !== 'function') {
    throw new Error('createBackgroundCache requires fetchPage');
  }

  const pages = new Map();
  const pending = new Map();

  const cacheKey = (page, limit) => `${page}:${limit}`;

  async function loadPage(page, { limit = 6, force = false } = {}) {
    const key = cacheKey(page, limit);
    if (!force && pages.has(key)) {
      return pages.get(key);
    }
    if (!force && pending.has(key)) {
      return pending.get(key);
    }

    const request = fetchPage({ page, limit }).then((result) => {
      const value = {
        items: result.items || result.prompts || [],
        total: result.total || 0,
        loadedAt: Date.now(),
      };
      pages.set(key, value);
      pending.delete(key);
      return value;
    }).catch((error) => {
      pending.delete(key);
      throw error;
    });

    pending.set(key, request);
    return request;
  }

  function invalidatePage(page, { limit = 6 } = {}) {
    pages.delete(cacheKey(page, limit));
  }

  function invalidateAll() {
    pages.clear();
    pending.clear();
  }

  return {
    loadPage,
    invalidatePage,
    invalidateAll,
  };
}
