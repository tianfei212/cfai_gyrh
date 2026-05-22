import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

function immutableModelCacheHeaders() {
  return {
    name: 'immutable-model-cache-headers',
    configureServer(server) {
      server.middlewares.use((req, res, next) => {
        if (req.url?.startsWith('/models/selfie_segmentation/')) {
          res.setHeader('Cache-Control', 'public, max-age=31536000, immutable');
        }
        next();
      });
    },
    configurePreviewServer(server) {
      server.middlewares.use((req, res, next) => {
        if (req.url?.startsWith('/models/selfie_segmentation/')) {
          res.setHeader('Cache-Control', 'public, max-age=31536000, immutable');
        }
        next();
      });
    },
  };
}

export default defineConfig({
  plugins: [react(), immutableModelCacheHeaders()],
  server: {
    host: '127.0.0.1',
    port: 9912,
    allowedHosts: true,
    hmr: {
      host: '127.0.0.1',
      protocol: 'ws',
    },
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:9913',
        changeOrigin: true,
        timeout: 120000,
        proxyTimeout: 120000,
      },
      '/images_data': {
        target: 'http://127.0.0.1:18080',
        changeOrigin: true,
      },
      '/gyrh_images_data': {
        target: 'http://127.0.0.1:18081',
        changeOrigin: true,
      },
    },
  },
});
