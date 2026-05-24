import { buildImageThumbnailUrl } from './imageThumbs.js';

const PROVIDER_STYLE_VALUES = new Set(['google', 'wan', '302-gpt-image']);

export function normalizeHistoryStyle(styleTransform = '') {
  const style = String(styleTransform || '').trim();
  return PROVIDER_STYLE_VALUES.has(style.toLowerCase()) ? '' : style;
}

export function mapGeneratedImagesToHistoryRecords(images = []) {
  return images.map((img) => ({
    id: img.id,
    url: buildImageThumbnailUrl({ assetId: img.asset_id, imageUrl: img.image_url }),
    rawUrl: img.image_url || `/api/v1/images/view?id=${img.id}`,
    assetId: img.asset_id || '',
    provider: img.provider || img.style_transform,
    style: normalizeHistoryStyle(img.style_transform),
    status: img.status,
    created_at: img.created_at,
    width: img.image_width || 0,
    height: img.image_height || 0,
  }));
}

export function buildHistoryTitle(total = 0) {
  return `历史记录 (${total})`;
}

export function buildHistoryPreviewPayload(record) {
  if (!record) {
    return null;
  }

  return {
    image: record.rawUrl || record.url,
    mode: 'single',
    assetId: record.assetId || '',
    style: record.style || '',
  };
}

export function getHistoryPageAfterDeletion({ page = 1, total = 0, deletedCount = 0, limit = 12 } = {}) {
  const remainingTotal = Math.max(0, total - deletedCount);
  const totalPages = Math.max(1, Math.ceil(remainingTotal / limit));

  return Math.min(Math.max(1, page), totalPages);
}
