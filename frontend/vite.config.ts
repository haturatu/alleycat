import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const publicDir = path.resolve(__dirname, "public");

const hasPublicAssets = () => {
  try {
    return fs.readdirSync(publicDir).some((entry) => !entry.startsWith("."));
  } catch {
    return false;
  }
};

export default defineConfig({
  publicDir: process.env.VITE_COPY_PUBLIC === "false" ? false : publicDir,
  plugins: [
    react(),
    {
      name: "theme-status",
      configureServer(server) {
        server.middlewares.use("/theme-status", (_req, res) => {
          res.setHeader("Content-Type", "application/json");
          res.setHeader("Cache-Control", "no-store");
          res.end(JSON.stringify({ publicAssets: hasPublicAssets() }));
        });
      },
      generateBundle() {
        this.emitFile({
          type: "asset",
          fileName: "theme-status",
          source: JSON.stringify({ publicAssets: hasPublicAssets() }),
        });
      },
    },
  ],
  resolve: {
    alias: {
      "@cms": path.resolve(__dirname, "src/cms"),
    },
  },
  base: process.env.VITE_BASE || "/",
  test: {
    environment: "jsdom",
  },
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
