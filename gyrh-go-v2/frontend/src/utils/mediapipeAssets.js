export const MEDIAPIPE_SELFIE_SEGMENTATION_BASE = '/models/selfie_segmentation';
export const MEDIAPIPE_SELFIE_SEGMENTATION_VERSION = '202605220122';

export const MEDIAPIPE_SELFIE_SEGMENTATION_FILES = [
  'selfie_segmentation.binarypb',
  'selfie_segmentation.js',
  'selfie_segmentation_solution_wasm_bin.js',
  'selfie_segmentation_solution_simd_wasm_bin.js',
  'selfie_segmentation_solution_wasm_bin.wasm',
  'selfie_segmentation_solution_simd_wasm_bin.wasm',
  'selfie_segmentation_solution_simd_wasm_bin.data',
  'selfie_segmentation.tflite',
  'selfie_segmentation_landscape.tflite',
];

export function getSelfieSegmentationAssetUrl(file) {
  return `${MEDIAPIPE_SELFIE_SEGMENTATION_BASE}/${file}?v=${MEDIAPIPE_SELFIE_SEGMENTATION_VERSION}`;
}

export function getSelfieSegmentationModelPath(modelSelection = 1) {
  const filename = modelSelection === 1
    ? 'selfie_segmentation_landscape.tflite'
    : 'selfie_segmentation.tflite';
  return getSelfieSegmentationAssetUrl(filename);
}

export function getSelfieSegmentationAssetUrls() {
  return MEDIAPIPE_SELFIE_SEGMENTATION_FILES.map((file) => getSelfieSegmentationAssetUrl(file));
}

export function prefetchSelfieSegmentationAssets(fetcher = globalThis.fetch) {
  if (typeof fetcher !== 'function') return [];
  return getSelfieSegmentationAssetUrls().map((url) => (
    fetcher(url, { cache: 'force-cache' }).catch(() => null)
  ));
}

let selfieSegmentationScriptPromise;
let mediapipeLogFilterInstalled = false;

export function isNoisyMediaPipeLog(args = []) {
  const message = args.map((item) => String(item)).join(' ');
  return (
    /[IW]\d{4} .*gl_context(_webgl)?\.cc:\d+/.test(message)
    || message.includes('Successfully created a WebGL context')
    || message.includes('GL version: 3.0')
    || message.includes('OpenGL error checking is disabled')
    || message.includes('Created TensorFlow Lite XNNPACK delegate for CPU')
  );
}

export function installMediaPipeLogFilter(targetConsole = globalThis.console) {
  if (mediapipeLogFilterInstalled || !targetConsole) return;
  const methods = ['log', 'info', 'warn'];
  for (const method of methods) {
    if (typeof targetConsole[method] !== 'function') continue;
    const original = targetConsole[method].bind(targetConsole);
    targetConsole[method] = (...args) => {
      if (isNoisyMediaPipeLog(args)) return;
      original(...args);
    };
  }
  mediapipeLogFilterInstalled = true;
}

export function loadSelfieSegmentationConstructor() {
  if (globalThis.SelfieSegmentation) {
    return Promise.resolve(globalThis.SelfieSegmentation);
  }
  if (!selfieSegmentationScriptPromise) {
    selfieSegmentationScriptPromise = new Promise((resolve, reject) => {
      if (typeof document === 'undefined') {
        reject(new Error('MediaPipe selfie segmentation requires a browser document'));
        return;
      }
      installMediaPipeLogFilter();
      const script = document.createElement('script');
      script.src = getSelfieSegmentationAssetUrl('selfie_segmentation.js');
      script.async = true;
      script.onload = () => {
        if (globalThis.SelfieSegmentation) {
          resolve(globalThis.SelfieSegmentation);
          return;
        }
        reject(new Error('MediaPipe SelfieSegmentation constructor was not registered'));
      };
      script.onerror = () => reject(new Error('Failed to load MediaPipe selfie_segmentation.js'));
      document.head.appendChild(script);
    });
  }
  return selfieSegmentationScriptPromise;
}
