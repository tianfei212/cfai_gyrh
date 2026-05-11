import test from 'node:test';
import assert from 'node:assert/strict';
import {
  getCameraScaleFromZoomOffset,
  getCenteredZoomCrop,
  getHorizontalMirrorTransform,
} from './cameraZoom.js';

test('maps UI zoom offset to a safe camera scale', () => {
  assert.equal(getCameraScaleFromZoomOffset(-1), 0.5);
  assert.equal(getCameraScaleFromZoomOffset(0), 1);
  assert.equal(getCameraScaleFromZoomOffset(1.5), 2.5);
});

test('builds a horizontal mirror transform from canvas width', () => {
  assert.deepEqual(getHorizontalMirrorTransform(1280), {
    translateX: 1280,
    scaleX: -1,
  });
});

test('uses the full frame when zoom is 1x', () => {
  assert.deepEqual(getCenteredZoomCrop({ width: 1280, height: 720, zoom: 1 }), {
    sx: 0,
    sy: 0,
    sw: 1280,
    sh: 720,
    dx: 0,
    dy: 0,
    dw: 1280,
    dh: 720,
  });
});

test('crops the center of the frame when zoomed in', () => {
  assert.deepEqual(getCenteredZoomCrop({ width: 1280, height: 720, zoom: 2 }), {
    sx: 320,
    sy: 180,
    sw: 640,
    sh: 360,
    dx: 0,
    dy: 0,
    dw: 1280,
    dh: 720,
  });
});

test('draws the full frame smaller when zoomed out', () => {
  assert.deepEqual(getCenteredZoomCrop({ width: 1280, height: 720, zoom: 0.5 }), {
    sx: 0,
    sy: 0,
    sw: 1280,
    sh: 720,
    dx: 320,
    dy: 180,
    dw: 640,
    dh: 360,
  });
});

test('clamps invalid zoom values to 1x', () => {
  assert.deepEqual(getCenteredZoomCrop({ width: 1280, height: 720, zoom: 0 }), {
    sx: 0,
    sy: 0,
    sw: 1280,
    sh: 720,
    dx: 0,
    dy: 0,
    dw: 1280,
    dh: 720,
  });
});
