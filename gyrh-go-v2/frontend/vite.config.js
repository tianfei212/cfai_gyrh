import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
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
