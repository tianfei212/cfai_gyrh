import test from 'node:test';
import assert from 'node:assert/strict';
import {
  LANDSCAPE_THUMB_HEIGHT,
  LANDSCAPE_THUMB_WIDTH,
  buildImageThumbnailUrl,
} from './imageThumbs.js';

test('builds 16:9 thumbnail URL from asset id', () => {
  assert.equal(LANDSCAPE_THUMB_WIDTH, 400);
  assert.equal(LANDSCAPE_THUMB_HEIGHT, 225);
  assert.equal(
    buildImageThumbnailUrl({ assetId: 'asset/one.png' }),
    '/api/v1/images/thumbnail?asset_id=asset%2Fone.png&w=400&h=225',
  );
});

test('builds 16:9 thumbnail URL from image URL when asset id is missing', () => {
  assert.equal(
    buildImageThumbnailUrl({ imageUrl: 'https://example.com/a b.png' }),
    '/api/v1/images/thumbnail?url=https%3A%2F%2Fexample.com%2Fa%20b.png&w=400&h=225',
  );
});

test('returns empty thumbnail URL without an image source', () => {
  assert.equal(buildImageThumbnailUrl({}), '');
});
