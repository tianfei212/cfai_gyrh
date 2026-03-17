
/**
 * Adds a watermark to an image.
 * @param imageBase64 The source image in Base64 format.
 * @param watermarkUrl The URL or path to the watermark image.
 * @param position 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left'
 * @param opacity Opacity of the watermark (0.0 - 1.0)
 * @param scaleRatio Ratio of watermark size relative to the image width (e.g., 0.2 for 20% width)
 * @returns Promise<string> Base64 string of the watermarked image.
 */
export const addWatermark = (
  imageBase64: string,
  watermarkUrl: string,
  position: 'bottom-right' | 'bottom-left' | 'top-right' | 'top-left' = 'bottom-right',
  opacity: number = 0.9,
  scaleRatio: number = 0.2
): Promise<string> => {
  return new Promise((resolve, reject) => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      const canvas = document.createElement('canvas');
      canvas.width = img.width;
      canvas.height = img.height;
      const ctx = canvas.getContext('2d');
      if (!ctx) {
        reject(new Error('Failed to get canvas context'));
        return;
      }

      // Draw original image
      ctx.drawImage(img, 0, 0);

      const watermark = new Image();
      watermark.crossOrigin = 'anonymous';
      watermark.onload = () => {
        // Calculate watermark size
        const wmWidth = img.width * scaleRatio;
        const wmHeight = (watermark.height / watermark.width) * wmWidth;

        // Calculate position
        let x = 0;
        let y = 0;
        const padding = img.width * 0.03; // 3% padding

        switch (position) {
          case 'bottom-right':
            x = img.width - wmWidth - padding;
            y = img.height - wmHeight - padding;
            break;
          case 'bottom-left':
            x = padding;
            y = img.height - wmHeight - padding;
            break;
          case 'top-right':
            x = img.width - wmWidth - padding;
            y = padding;
            break;
          case 'top-left':
            x = padding;
            y = padding;
            break;
        }

        // Draw watermark
        ctx.globalAlpha = opacity;
        ctx.drawImage(watermark, x, y, wmWidth, wmHeight);
        ctx.globalAlpha = 1.0;

        // Return result
        resolve(canvas.toDataURL('image/png'));
      };
      watermark.onerror = (e) => {
        console.warn('Failed to load watermark image, returning original.', e);
        resolve(imageBase64); // Return original if watermark fails
      };
      watermark.src = watermarkUrl;
    };
    img.onerror = (e) => reject(e);
    img.src = imageBase64;
  });
};
