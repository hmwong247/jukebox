import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

const DEV_ORIGIN = "http://localhost:8080"

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      "/api": {
        target: DEV_ORIGIN,
        changeOrigin: true,
      },
      "/ws": {
        target: "ws://localhost:8080",
        changeOrigin: true,
        ws: true,
        rewriteWsOrigin: true,
      },
      // "/home": {
      //   target: DEV_ORIGIN,
      //   changeOrigin: true,
      // },
      // "/join": {
      //   target: DEV_ORIGIN,
      //   changeOrigin: true,
      // },
    }
  }
})
