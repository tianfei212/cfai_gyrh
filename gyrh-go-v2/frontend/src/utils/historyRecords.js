import { buildImageThumbnailUrl } from './imageThumbs.js';

export function mapGeneratedImagesToHistoryRecords(images = []) {
  return images.map((img) => ({
    id: img.id,
    url: buildImageThumbnailUrl({ assetId: img.asset_id, imageUrl: img.image_url }),
    rawUrl: img.image_url || `/api/v1/images/view?id=${img.id}`,
    provider: img.provider || img.style_transform,
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
  };
}

export function getHistoryPageAfterDeletion({ page = 1, total = 0, deletedCount = 0, limit = 12 } = {}) {
  const remainingTotal = Math.max(0, total - deletedCount);
  const totalPages = Math.max(1, Math.ceil(remainingTotal / limit));

  return Math.min(Math.max(1, page), totalPages);
}
