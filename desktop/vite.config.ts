import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { VitePlugin } from "@tauri-apps/cli/vite";
import { fileURLToPath, URL } from "node:url";

// The dev proxy lets the Vite dev server and the Tauri webview talk to a
// locally running Go backend on its canonical address.
const BACKEND = "http://127.0.0.1:8080";

export default defineConfig({
  plugins: [react(), VitePlugin()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  clearScreen: false,
  server: {
    port: 1420,
    strictPort: true,
    watch: {
      ignored: ["**/src-tauri/**"],
    },
    proxy: {
      "/api": { target: BACKEND, changeOrigin: true },
      "/uploads": { target: BACKEND, changeOrigin: true },
    },
  },
});