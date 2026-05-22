import test from 'node:test';
import assert from 'node:assert/strict';
import { normalizePreviewSelection } from './previewSelection.js';

test('normalizes plain image values as compare preview mode', () => {
  assert.deepEqual(normalizePreviewSelection('/image.png'), {
    image: '/image.png',
    mode: 'compare',
    assetId: '',
  });
});

test('preserves explicit single preview mode from history records', () => {
  assert.deepEqual(normalizePreviewSelection({ image: '/history.png', mode: 'single', assetId: 'generated:one.png' }), {
    image: '/history.png',
    mode: 'single',
    assetId: 'generated:one.png',
  });
});
