import { fileURLToPath } from 'node:url'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig, loadEnv } from 'vite'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const vendorTabsFile = fileURLToPath(
    new URL(
      env.VITE_HAS_VENDOR_MODEM === 'false'
        ? './src/components/tabs/vendor_tabs.nomodem.ts'
        : './src/components/tabs/vendor_tabs.ts',
      import.meta.url
    )
  )

  return {
    base: './',
    plugins: [svelte()],
    resolve: {
      alias: [{ find: './vendor_tabs', replacement: vendorTabsFile }],
    },
    server: {
      port: 5173,
      host: true,
      proxy: {
        '/api': {
          target: 'http://127.0.0.1:80',
          changeOrigin: true,
          ws: true,
        },
      },
    },
    build: {
      outDir: 'dist',
      emptyOutDir: true,
      rollupOptions: {
        output: {
          // NOTE: [hash] alone is enough for cache-busting. A timestamp was
          // removed here to keep builds reproducible (identical content must
          // produce identical filenames, otherwise every release churns).
          entryFileNames: `assets/[name]-[hash].js`,
          chunkFileNames: `assets/[name]-[hash].js`,
          assetFileNames: `assets/[name]-[hash].[ext]`,
          manualChunks(id) {
            if (id.includes('@xterm')) {
              return 'xterm'
            }
            if (id.includes('@lucide')) {
              return 'lucide'
            }
          },
        },
      },
    },
  }
})
