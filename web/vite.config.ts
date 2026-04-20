import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const base = process.env.VITE_BASE_PATH || "/";

export default defineConfig({
  base: base === "/" ? "/" : base.endsWith("/") ? base : `${base}/`,
  plugins: [react()],
  server: {
    proxy: {
      "/v1": "http://127.0.0.1:8080",
      "/healthz": "http://127.0.0.1:8080",
      // 本地用 VITE_BASE_PATH=/skip/ 试子路径时，API 仍转发到本机后端根路径
      "/skip/v1": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        rewrite: (p) => p.replace(/^\/skip/, ""),
      },
      "/skip/healthz": {
        target: "http://127.0.0.1:8080",
        changeOrigin: true,
        rewrite: () => "/healthz",
      },
    },
  },
});
