export const LANDSCAPE_THUMB_WIDTH = 400;
export const LANDSCAPE_THUMB_HEIGHT = 225;
export const CAPTURE_BACKGROUND_WIDTH = 1280;
export const CAPTURE_BACKGROUND_HEIGHT = 720;
export const IMAGE_CACHE_BUCKET_MS = 3 * 60 * 1000;

export function buildImageThumbnailUrl({ assetId, imageUrl } = {}) {
  const size = `w=${LANDSCAPE_THUMB_WIDTH}&h=${LANDSCAPE_THUMB_HEIGHT}`;
  return buildThumbnailUrl({ assetId, imageUrl, size });
}

export function buildCaptureBackgroundThumbnailUrl({ assetId, imageUrl } = {}) {
  const size = `w=${CAPTURE_BACKGROUND_WIDTH}&h=${CAPTURE_BACKGROUND_HEIGHT}`;
  return buildThumbnailUrl({ assetId, imageUrl, size });
}

export function getImagePreloadUrls(items = []) {
  const urls = new Set();
  for (const item of items) {
    const url = buildImageThumbnailUrl({
      assetId: item?.image_asset_id,
      imageUrl: item?.image_url,
    });
    if (url) {
      urls.add(url);
    }
  }
  return Array.from(urls);
}

export function preloadImages(urls = []) {
  if (typeof Image === 'undefined') return;
  for (const url of urls) {
    if (!url) continue;
    const image = new Image();
    image.src = url;
  }
}

export function getImageCacheBucket(now = Date.now()) {
  return Math.floor(now / IMAGE_CACHE_BUCKET_MS);
}

export function appendImageCacheBucket(url, bucket = getImageCacheBucket()) {
  if (!url) return '';
  const separator = url.includes('?') ? '&' : '?';
  return `${url}${separator}rv=${encodeURIComponent(bucket)}`;
}

export function refreshImageUrl(url, retryToken = Date.now()) {
  if (!url) return '';
  const withoutRetry = url.replace(/([?&])retry=[^&]*&?/, '$1').replace(/[?&]$/, '');
  const separator = withoutRetry.includes('?') ? '&' : '?';
  return `${withoutRetry}${separator}retry=${encodeURIComponent(retryToken)}`;
}

function buildThumbnailUrl({ assetId, imageUrl, size }) {
  let url = '';
  if (assetId) {
    url = `/api/v1/images/thumbnail?asset_id=${encodeURIComponent(assetId)}&${size}`;
  } else if (imageUrl) {
    url = `/api/v1/images/thumbnail?url=${encodeURIComponent(imageUrl)}&${size}`;
  }
  return appendImageCacheBucket(url);
}
