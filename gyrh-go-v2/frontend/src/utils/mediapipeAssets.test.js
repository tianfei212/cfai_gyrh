import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
import { join } from 'node:path';
import test from 'node:test';
import {
  MEDIAPIPE_SELFIE_SEGMENTATION_FILES,
  MEDIAPIPE_SELFIE_SEGMENTATION_VERSION,
  getSelfieSegmentationAssetUrl,
  getSelfieSegmentationModelPath,
  isNoisyMediaPipeLog,
  loadSelfieSegmentationConstructor,
} from './mediapipeAssets.js';

test('selfie segmentation local model assets are present', () => {
  const modelDir = join(process.cwd(), 'public', 'models', 'selfie_segmentation');

  for (const filename of MEDIAPIPE_SELFIE_SEGMENTATION_FILES) {
    assert.equal(existsSync(join(modelDir, filename)), true, `${filename} should exist`);
  }
});

test('landscape model path matches MediaPipe modelSelection 1', () => {
  assert.equal(
    getSelfieSegmentationModelPath(1),
    `/models/selfie_segmentation/selfie_segmentation_landscape.tflite?v=${MEDIAPIPE_SELFIE_SEGMENTATION_VERSION}`,
  );
});

test('MediaPipe asset URLs are versioned to bypass stale immutable caches', () => {
  assert.equal(
    getSelfieSegmentationAssetUrl('selfie_segmentation_solution_wasm_bin.wasm'),
    `/models/selfie_segmentation/selfie_segmentation_solution_wasm_bin.wasm?v=${MEDIAPIPE_SELFIE_SEGMENTATION_VERSION}`,
  );
});

test('uses an existing global selfie segmentation constructor when available', async () => {
  const previous = globalThis.SelfieSegmentation;
  function FakeSelfieSegmentation() {}
  globalThis.SelfieSegmentation = FakeSelfieSegmentation;

  try {
    assert.equal(await loadSelfieSegmentationConstructor(), FakeSelfieSegmentation);
  } finally {
    if (previous) {
      globalThis.SelfieSegmentation = previous;
    } else {
      delete globalThis.SelfieSegmentation;
    }
  }
});

test('identifies noisy MediaPipe native initialization logs', () => {
  assert.equal(
    isNoisyMediaPipeLog(['I0000 00:00:1779375712.116000       1 gl_context_webgl.cc:151] Successfully created a WebGL context with major version 3 and handle 3']),
    true,
  );
  assert.equal(
    isNoisyMediaPipeLog(['W0000 00:00:1779375712.119000       1 gl_context.cc:1000] OpenGL error checking is disabled']),
    true,
  );
  assert.equal(
    isNoisyMediaPipeLog(['INFO: Created TensorFlow Lite XNNPACK delegate for CPU.']),
    true,
  );
  assert.equal(isNoisyMediaPipeLog(['Failed to initialize MediaPipe selfie segmentation']), false);
});
