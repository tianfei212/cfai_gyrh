import test from 'node:test';
import assert from 'node:assert/strict';
import {
  buildHistoryPreviewPayload,
  buildHistoryTitle,
  getHistoryPageAfterDeletion,
  mapGeneratedImagesToHistoryRecords,
} from './historyRecords.js';
import { IMAGE_CACHE_BUCKET_MS } from './imageThumbs.js';

const originalDateNow = Date.now;

test.beforeEach(() => {
  Date.now = () => IMAGE_CACHE_BUCKET_MS * 7;
});

test.afterEach(() => {
  Date.now = originalDateNow;
});

test('maps generated images to newest-first history records without reordering', () => {
  const records = mapGeneratedImagesToHistoryRecords([
    {
      id: 74,
      asset_id: 'generated:newest.png',
      image_url: 'https://example.com/newest.png',
      provider: 'wan',
      style_transform: '水彩电影',
      status: 'succeeded',
      created_at: '2026-05-11T14:00:00Z',
      image_width: 1080,
      image_height: 1920,
    },
    {
      id: 73,
      asset_id: 'generated:older.png',
      image_url: 'https://example.com/older.png',
      style_transform: 'google',
      status: 'succeeded',
      created_at: '2026-05-11T13:00:00Z',
    },
  ]);

  assert.deepEqual(
    records.map((record) => record.id),
    [74, 73],
  );
  assert.equal(
    records[0].url,
    '/api/v1/images/thumbnail?asset_id=generated%3Anewest.png&w=400&h=225&rv=7',
  );
  assert.equal(records[0].assetId, 'generated:newest.png');
  assert.equal(records[0].rawUrl, 'https://example.com/newest.png');
  assert.equal(records[0].style, '水彩电影');
  assert.equal(records[1].provider, 'google');
  assert.equal(records[1].style, '');
  assert.equal(records[1].width, 0);
  assert.equal(records[1].height, 0);
});

test('builds history title from real database total', () => {
  assert.equal(buildHistoryTitle(74), '历史记录 (74)');
});

test('builds single-image preview payload from a history record', () => {
  assert.deepEqual(
    buildHistoryPreviewPayload({
      id: 1,
      url: '/api/v1/images/thumbnail?asset_id=one&w=400&h=225',
      rawUrl: '/api/v1/images/view?id=1',
      assetId: 'generated:one.png',
      style: '漫画风格',
    }),
    {
      image: '/api/v1/images/view?id=1',
      mode: 'single',
      assetId: 'generated:one.png',
      style: '漫画风格',
    },
  );
});

test('calculates the page to refresh after selected records are deleted', () => {
  assert.equal(getHistoryPageAfterDeletion({ page: 2, total: 13, deletedCount: 1, limit: 12 }), 1);
  assert.equal(getHistoryPageAfterDeletion({ page: 2, total: 14, deletedCount: 1, limit: 12 }), 2);
  assert.equal(getHistoryPageAfterDeletion({ page: 1, total: 3, deletedCount: 3, limit: 12 }), 1);
});
