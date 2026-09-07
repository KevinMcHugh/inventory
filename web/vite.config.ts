import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/tenant": "http://localhost:8080",
      "/kinds": "http://localhost:8080",
      "/health": "http://localhost:8080",
    },
  },
});
