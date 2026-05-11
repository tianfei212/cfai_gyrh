export function normalizePreviewSelection(selection) {
  if (selection && typeof selection === 'object') {
    return {
      image: selection.image,
      mode: selection.mode === 'single' ? 'single' : 'compare',
    };
  }

  return {
    image: selection,
    mode: 'compare',
  };
}
