import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

const DEV_ORIGIN = "http://localhost:8080"

// https://vite.dev/config/
export default defineConfig({
  plugins: [svelte(), tailwindcss()],
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
  },
  resolve: {
    alias: {
      "simple-peer": "simple-peer/simplepeer.min.js",
      "@scripts": path.resolve(__dirname, "src/scripts"),
      "@components": path.resolve(__dirname, "src/components"),
    },
  },
})
