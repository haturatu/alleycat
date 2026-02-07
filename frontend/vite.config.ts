import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  base: process.env.VITE_BASE || "/",
  server: {
    host: "0.0.0.0",
    allowedHosts: "all",
    proxy: {
      ...(process.env.VITE_PB_PROXY_TARGET
        ? {
            "/api": {
              target: process.env.VITE_PB_PROXY_TARGET,
              changeOrigin: true,
              secure: false,
            },
          }
        : {}),
      ...(process.env.VITE_UPLOADS_PROXY_TARGET
        ? {
            "/uploads": {
              target: process.env.VITE_UPLOADS_PROXY_TARGET,
              changeOrigin: true,
              secure: false,
            },
          }
        : {}),
    },
  },
});
