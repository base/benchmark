import { defineConfig, loadEnv } from "vite";
import { resolve } from "path";
import { viteStaticCopy } from "vite-plugin-static-copy";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const isLocal = env.IS_LOCAL === "true";

  const processEnvVars = {
    API_BASE_URL: env.API_BASE_URL,
    VITE_API_BASE_URL: env.API_BASE_URL,
  }

  // Parse allowed hosts from environment variable (comma-separated list)
  const allowedHosts = env.VITE_ALLOWED_HOSTS 
    ? env.VITE_ALLOWED_HOSTS.split(',').map(host => host.trim())
    : ['localhost'];

  return {
    server: {
      port: 3000,
    },
    preview: {
      port: 3000,
      host: "0.0.0.0",
      allowedHosts: allowedHosts,
    },
    base: "./", // use relative path for github pages
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
    define: {
      // Define process.env to avoid undefined errors
      'process.env': JSON.stringify(processEnvVars),
      // Also define global process if needed
      process: JSON.stringify({
        env: processEnvVars,
      }),
    },
    // assetsInclude: ['../output/**/*']
    plugins: [
      react(),
      tailwindcss(),
      isLocal && viteStaticCopy({
        targets: [
          {
            src: "../output/**/*",
            dest: "output",
          },
        ],
      }),
    ]
  }
});
