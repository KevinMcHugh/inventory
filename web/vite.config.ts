import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/tenant": "http://localhost:8080",
      "/kinds": "http://localhost:8080",
      "/mcp/rpc": "http://localhost:8080",
      "/oauth": "http://localhost:8080",
      "/.well-known": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
});
