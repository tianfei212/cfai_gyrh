import test from 'node:test';
import assert from 'node:assert/strict';
import {
  IMAGE_CACHE_BUCKET_MS,
  LANDSCAPE_THUMB_HEIGHT,
  LANDSCAPE_THUMB_WIDTH,
  appendImageCacheBucket,
  buildCaptureBackgroundThumbnailUrl,
  buildFullImagePreviewUrl,
  buildImageThumbnailUrl,
  getImageCacheBucket,
  getImagePreloadUrls,
  refreshImageUrl,
} from './imageThumbs.js';

const originalDateNow = Date.now;

test.beforeEach(() => {
  Date.now = () => IMAGE_CACHE_BUCKET_MS * 42 + 1000;
});

test.afterEach(() => {
  Date.now = originalDateNow;
});

test('builds 16:9 thumbnail URL from asset id', () => {
  assert.equal(LANDSCAPE_THUMB_WIDTH, 400);
  assert.equal(LANDSCAPE_THUMB_HEIGHT, 225);
  assert.equal(
    buildImageThumbnailUrl({ assetId: 'asset/one.png' }),
    '/api/v1/images/thumbnail?asset_id=asset%2Fone.png&w=400&h=225&rv=42',
  );
});

test('builds 16:9 thumbnail URL from image URL when asset id is missing', () => {
  assert.equal(
    buildImageThumbnailUrl({ imageUrl: 'https://example.com/a b.png' }),
    '/api/v1/images/thumbnail?url=https%3A%2F%2Fexample.com%2Fa%20b.png&w=400&h=225&rv=42',
  );
});

test('returns empty thumbnail URL without an image source', () => {
  assert.equal(buildImageThumbnailUrl({}), '');
});

test('builds full image preview URL preferring OSS image URL', () => {
  assert.equal(
    buildFullImagePreviewUrl({
      assetId: 'asset/one.png',
      imageUrl: 'https://example.com/full.webp',
    }),
    'https://example.com/full.webp',
  );
});

test('builds full image preview URL from asset id when image URL is missing', () => {
  assert.equal(
    buildFullImagePreviewUrl({ assetId: 'asset/one.png' }),
    '/api/v1/images/view?asset_id=asset%2Fone.png&rv=42',
  );
});

test('builds capture background thumbnail URL for selected backgrounds', () => {
  assert.equal(
    buildCaptureBackgroundThumbnailUrl({ imageUrl: 'https://example.com/bg.png' }),
    '/api/v1/images/thumbnail?url=https%3A%2F%2Fexample.com%2Fbg.png&w=1280&h=720&rv=42',
  );
});

test('image cache bucket rolls every three minutes', () => {
  assert.equal(getImageCacheBucket(0), 0);
  assert.equal(getImageCacheBucket(IMAGE_CACHE_BUCKET_MS - 1), 0);
  assert.equal(getImageCacheBucket(IMAGE_CACHE_BUCKET_MS), 1);
  assert.equal(appendImageCacheBucket('/x?a=1', 9), '/x?a=1&rv=9');
});

test('refreshes an image URL immediately after a failed load', () => {
  assert.equal(refreshImageUrl('/x?a=1&rv=9', 123), '/x?a=1&rv=9&retry=123');
  assert.equal(refreshImageUrl('/x?a=1&rv=9&retry=old', 124), '/x?a=1&rv=9&retry=124');
});

test('deduplicates image URLs for preloading', () => {
  assert.deepEqual(
    getImagePreloadUrls([
      { image_url: 'https://example.com/a.png' },
      { image_url: 'https://example.com/a.png' },
      { image_url: 'https://example.com/b.png' },
      {},
    ]),
    [
      '/api/v1/images/thumbnail?url=https%3A%2F%2Fexample.com%2Fa.png&w=400&h=225&rv=42',
      '/api/v1/images/thumbnail?url=https%3A%2F%2Fexample.com%2Fb.png&w=400&h=225&rv=42',
    ],
  );
});
