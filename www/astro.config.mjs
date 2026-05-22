import { defineConfig } from "astro/config";

export default defineConfig({
  site: "https://fluxlb.dev",
  outDir: "./dist",
  publicDir: "./public",
  build: {
    format: "directory",
  },
});
