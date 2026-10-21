import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import { TanStackRouterVite } from "@tanstack/router-plugin/vite";
import tailwindcss from "@tailwindcss/vite";
import tsconfigPaths from "vite-tsconfig-paths";
import postcssPresetEnv from "postcss-preset-env";
import { resolve } from "path";
import { resolveViteHost } from "./src/lib/viteHostPolicy";

export default defineConfig(() => ({
  base: "/",

  plugins: [
    TanStackRouterVite({ target: "react" }),
    react(),
    tailwindcss(),
    tsconfigPaths(),
  ],

  resolve: {
    alias: {
      "@/gotypes": resolve(__dirname, "codegen/gotypes.gen.ts"),
      "@": resolve(__dirname, "src"),
      "micromark-extension-math": "micromark-extension-llm-math",
    },
  },

  css: {
    postcss: {
      plugins: [
        postcssPresetEnv({
          stage: 1, // Include more experimental features that Safari 14 needs
          browsers: ["Safari >= 14"],
          // autoprefixer: false,
          features: {
            "custom-properties": true, // Let TailwindCSS handle this
            "nesting-rules": true,
            "logical-properties-and-values": true, // Polyfill logical properties
            "media-query-ranges": true, // Modern media query syntax
            "color-function": true, // CSS color functions
            "double-position-gradients": true,
            "gap-properties": true, // This is key for flexbox gap!
            "place-properties": true,
            "overflow-property": true,
            "focus-visible-pseudo-class": true, // Focus-visible support
            "focus-within-pseudo-class": true, // Focus-within support
            "any-link-pseudo-class": true, // :any-link pseudo-class
            "not-pseudo-class": true, // Enhanced :not() support
            "dir-pseudo-class": true, // :dir() pseudo-class
            "all-property": true, // CSS 'all' property
            "image-set-function": true, // image-set() function
            "hwb-function": true, // hwb() color function
            "lab-function": true, // lab() color function
            "oklab-function": true, // oklab() color function
          },
        }),
      ],
    },
  },

  build: {
    target: "es2017",
    rollupOptions: {
      output: {
        manualChunks(id) {
          // Os catalogos de i18n sao dados puros e grandes (dois dicionarios completos,
          // ~194 KB de fonte). No chunk principal empurraram o maior asset para 2.044.042 B,
          // acima do orcamento de 2.000.000 B: "Bundle budget exceeded by index-*.js"
          // (medido no CI, job "Web and mobile quality", e localmente com
          // `node scripts/check-bundle-budget.mjs`).
          // Nao se sobe o tecto: separa-se o que e' dado. Assim o chunk principal volta
          // para baixo do limite e os catalogos ficam num ficheiro proprio, cacheavel.
          // O import continua SINCRONO de proposito: `translate("en", ...)` tem de
          // responder no mesmo tick (o teste de paridade depende disso).
          if (/[\\/]src[\\/]i18n[\\/]messages\./.test(id)) return "i18n-catalogs";
          if (!id.includes("node_modules")) return undefined;
          if (/[\\/]react(?:-dom)?[\\/]|[\\/]scheduler[\\/]/.test(id)) return "vendor-react";
          if (id.includes("@tanstack")) return "vendor-router";
          if (/(heroicons|simple-icons)/.test(id)) return "vendor-icons";
          // Do not force the remaining dependency graph into a catch-all
          // chunk: React-heavy markdown packages can otherwise create a
          // circular chunk graph and execute before createContext exists.
          return undefined;
        },
      },
    },
  },

  esbuild: {
    target: "es2017",
  },

  server: {
    host: resolveViteHost(),
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.OLLAMA_PROXY_TARGET || "http://127.0.0.1:11434",
        changeOrigin: true,
      },
    },
  },
  preview: {
    host: resolveViteHost(),
    port: 5173,
    proxy: {
      "/api": {
        target: process.env.OLLAMA_PROXY_TARGET || "http://127.0.0.1:11434",
        changeOrigin: true,
      },
    },
  },
}));
