import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const currentDir = dirname(fileURLToPath(import.meta.url));
const sourceRoot = resolve(currentDir, '..');

function readSource(relativePath) {
  return readFileSync(resolve(sourceRoot, relativePath), 'utf8');
}

const styles = readSource('styles.css');
const mainEntry = readSource('main.jsx');
const kioskStyles = readSource('theme/kiosk.css');
const previewScreen = readSource('screens/PreviewScreen.jsx');
const styleManagerScreen = readSource('screens/StyleManagerScreen.jsx');

test('main entry loads the liquid glass skin', () => {
  assert.match(mainEntry, /import\s+['"]\.\/theme\/liquid-glass\.css['"]/);
});

test('global layout uses dynamic viewport units for fullscreen adaptation', () => {
  assert.match(styles, /--app-viewport-height:\s*100dvh/);
  assert.match(styles, /\.app-shell[\s\S]*?min-height:\s*var\(--app-viewport-height\)/);
});

test('kiosk layout disables fixed 16:9 framing on narrow or portrait screens', () => {
  assert.match(kioskStyles, /@media\s*\([^)]*max-width:\s*1180px[^)]*\),\s*\([^)]*orientation:\s*portrait[^)]*\)/);
  assert.match(kioskStyles, /aspect-ratio:\s*auto/);
});

test('capture controls have a low-height responsive layout', () => {
  assert.match(styles, /@media\s*\([^)]*max-height:\s*760px[^)]*\)/);
  assert.match(styles, /\.screen-capture\s+\.opacity-slider-wrapper/);
  assert.match(styles, /\.screen-capture\s+\.vertical-slider[\s\S]*?transform:\s*none/);
});

test('preview comparison layout is controlled by responsive classes', () => {
  assert.match(previewScreen, /preview-stage-container/);
  assert.doesNotMatch(previewScreen, /display:\s*'flex'[\s\S]*?gap:\s*'20px'/);
  assert.match(styles, /\.preview-stage-container[\s\S]*?display:\s*flex/);
  assert.match(styles, /@media\s*\([^)]*max-width:\s*900px[^)]*\)[\s\S]*?\.preview-stage-container[\s\S]*?flex-direction:\s*column/);
});

test('admin manager tables avoid inline fixed grid columns', () => {
  assert.doesNotMatch(styleManagerScreen, /gridTemplateColumns:\s*'80px 200px 100px 180px 1fr'/);
  assert.match(styles, /\.style-table-grid/);
});
