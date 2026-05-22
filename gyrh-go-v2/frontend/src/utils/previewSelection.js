export function normalizePreviewSelection(selection) {
  if (selection && typeof selection === 'object') {
    return {
      image: selection.image,
      mode: selection.mode === 'single' ? 'single' : 'compare',
      assetId: selection.assetId || '',
    };
  }

  return {
    image: selection,
    mode: 'compare',
    assetId: '',
  };
}
