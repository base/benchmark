import { defineConfig } from "vite";
import { resolve } from "path";
import { viteStaticCopy } from "vite-plugin-static-copy";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  server: {
    port: 3000,
  },
  build: {
    outDir: "dist",
    assetsDir: "assets",
    emptyOutDir: true,
    rollupOptions: {
      input: {
        main: resolve(__dirname, "index.html"),
      },
    },
  },
  // assetsInclude: ['../output/**/*']
  plugins: [
    react(),
    tailwindcss(),
    viteStaticCopy({
      targets: [
        {
          src: "../output/**/*",
          dest: "output",
        },
      ],
    }),
  ],
});
