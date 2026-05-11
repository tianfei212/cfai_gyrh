export const LANDSCAPE_THUMB_WIDTH = 400;
export const LANDSCAPE_THUMB_HEIGHT = 225;

export function buildImageThumbnailUrl({ assetId, imageUrl } = {}) {
  const size = `w=${LANDSCAPE_THUMB_WIDTH}&h=${LANDSCAPE_THUMB_HEIGHT}`;
  if (assetId) {
    return `/api/v1/images/thumbnail?asset_id=${encodeURIComponent(assetId)}&${size}`;
  }
  if (imageUrl) {
    return `/api/v1/images/thumbnail?url=${encodeURIComponent(imageUrl)}&${size}`;
  }
  return '';
}
