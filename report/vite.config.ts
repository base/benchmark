import { defineConfig, loadEnv } from "vite";
import { resolve } from "path";
import { existsSync } from "fs";
import { viteStaticCopy } from "vite-plugin-static-copy";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");

  const processEnvVars = {
    API_BASE_URL: env.VITE_API_BASE_URL,
    DATA_SOURCE: env.VITE_DATA_SOURCE,
    STATIC_BASE_PATH: env.VITE_STATIC_BASE_PATH,
  };

  // Parse allowed hosts from environment variable (comma-separated list)
  const allowedHosts = env.VITE_ALLOWED_HOSTS
    ? env.VITE_ALLOWED_HOSTS.split(",").map((host) => host.trim())
    : ["localhost"];

  // Check if output directory exists
  const outputDirPath = resolve(__dirname, "../output");
  const hasOutputDir = existsSync(outputDirPath);

  // Build plugins array
  const plugins = [react(), tailwindcss()];

  // Only add static copy plugin if output directory exists
  if (hasOutputDir) {
    plugins.push(
      viteStaticCopy({
        targets: [
          {
            src: "../output/**/*",
            dest: "output",
          },
        ],
      })
    );
  } else {
    console.log("Output directory not found, skipping static copy");
  }

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
      "process.env": JSON.stringify(processEnvVars),
      // Also define global process if needed
      process: JSON.stringify({
        env: processEnvVars,
      }),
    },
    plugins,
  };
});
