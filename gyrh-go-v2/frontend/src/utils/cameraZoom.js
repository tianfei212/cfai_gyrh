export function getCameraScaleFromZoomOffset(offset) {
  const safeOffset = Number.isFinite(offset) ? offset : 0;
  return safeOffset < 0 ? 1 / (1 + Math.abs(safeOffset)) : 1 + safeOffset;
}

export function getHorizontalMirrorTransform(width) {
  const safeWidth = Number.isFinite(width) && width > 0 ? width : 0;
  return {
    translateX: safeWidth,
    scaleX: -1,
  };
}

export function getCenteredZoomCrop({ width, height, zoom }) {
  const safeWidth = Number.isFinite(width) && width > 0 ? width : 0;
  const safeHeight = Number.isFinite(height) && height > 0 ? height : 0;
  const safeZoom = Number.isFinite(zoom) && zoom > 0 ? zoom : 1;

  if (safeZoom < 1) {
    const dw = safeWidth * safeZoom;
    const dh = safeHeight * safeZoom;

    return {
      sx: 0,
      sy: 0,
      sw: safeWidth,
      sh: safeHeight,
      dx: (safeWidth - dw) / 2,
      dy: (safeHeight - dh) / 2,
      dw,
      dh,
    };
  }

  const sw = safeWidth / safeZoom;
  const sh = safeHeight / safeZoom;

  return {
    sx: (safeWidth - sw) / 2,
    sy: (safeHeight - sh) / 2,
    sw,
    sh,
    dx: 0,
    dy: 0,
    dw: safeWidth,
    dh: safeHeight,
  };
}
