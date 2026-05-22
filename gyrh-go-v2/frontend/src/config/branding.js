import { useEffect, useState } from 'react';

export const BRANDING_CONFIG_URL = '/branding/brand-config.json';
export const PUBLIC_BRANDING_BASE = '/branding/';

export const DEFAULT_BRANDING = {
  appName: '中影AI 光影融合',
  productName: 'AI Smart Portrait',
  logo: '',
  background: '',
  previewWatermark: {
    brand: 'GYRH',
    product: 'AI PORTRAIT',
  },
};

function normalizeBrandingAssetPath(path) {
  if (!path || typeof path !== 'string') return '';
  if (path.startsWith('/') || path.startsWith('http://') || path.startsWith('https://')) {
    return path;
  }
  return `${PUBLIC_BRANDING_BASE}${path}`;
}

export function mergeBrandingConfig(config) {
  const source = config && typeof config === 'object' ? config : {};
  const watermark = source.previewWatermark && typeof source.previewWatermark === 'object'
    ? source.previewWatermark
    : {};

  return {
    ...DEFAULT_BRANDING,
    ...source,
    logo: normalizeBrandingAssetPath(source.logo ?? DEFAULT_BRANDING.logo),
    background: normalizeBrandingAssetPath(source.background ?? DEFAULT_BRANDING.background),
    previewWatermark: {
      ...DEFAULT_BRANDING.previewWatermark,
      ...watermark,
    },
  };
}

export function useBrandingConfig() {
  const [branding, setBranding] = useState(DEFAULT_BRANDING);

  useEffect(() => {
    let isMounted = true;

    async function loadBranding() {
      try {
        const response = await fetch(BRANDING_CONFIG_URL, { cache: 'no-cache' });
        if (!response.ok) return;
        const config = await response.json();
        if (isMounted) {
          setBranding(mergeBrandingConfig(config));
        }
      } catch (err) {
        console.warn('Failed to load branding config:', err);
      }
    }

    loadBranding();

    return () => {
      isMounted = false;
    };
  }, []);

  return branding;
}

export function buildScreenTitle(branding, title) {
  const productName = branding?.productName || DEFAULT_BRANDING.productName;
  return title ? `${productName} · ${title}` : productName;
}
