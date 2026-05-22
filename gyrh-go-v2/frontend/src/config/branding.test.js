import assert from 'node:assert/strict';
import { test } from 'node:test';
import { DEFAULT_BRANDING, mergeBrandingConfig } from './branding.js';

test('uses default branding when remote config is empty', () => {
  assert.deepEqual(mergeBrandingConfig(null), DEFAULT_BRANDING);
});

test('merges configured brand fields without dropping defaults', () => {
  const branding = mergeBrandingConfig({
    appName: '展厅光影系统',
    logo: 'logo.png',
    background: 'backgrounds/home.jpg',
    previewWatermark: {
      brand: 'CUSTOM',
    },
  });

  assert.equal(branding.appName, '展厅光影系统');
  assert.equal(branding.logo, '/branding/logo.png');
  assert.equal(branding.background, '/branding/backgrounds/home.jpg');
  assert.equal(branding.previewWatermark.brand, 'CUSTOM');
  assert.equal(branding.previewWatermark.product, DEFAULT_BRANDING.previewWatermark.product);
});
