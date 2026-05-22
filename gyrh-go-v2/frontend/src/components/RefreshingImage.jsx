import React, { useEffect, useState } from 'react';
import { refreshImageUrl } from '../utils/imageThumbs';

export function RefreshingImage({ src, maxRetries = 2, onError, ...props }) {
  const [currentSrc, setCurrentSrc] = useState(src || '');
  const [retryCount, setRetryCount] = useState(0);

  useEffect(() => {
    setCurrentSrc(src || '');
    setRetryCount(0);
  }, [src]);

  const handleError = (event) => {
    if (currentSrc && retryCount < maxRetries) {
      setRetryCount((count) => count + 1);
      setCurrentSrc(refreshImageUrl(currentSrc));
      return;
    }
    onError?.(event);
  };

  return <img {...props} src={currentSrc} onError={handleError} />;
}
