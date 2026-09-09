/** @type {import('tailwindcss').Config} */
export default {
  darkMode: ['class', '[data-theme="dark"]'],
  content: [
    './index.html',
    './src/**/*.{svelte,js,ts,jsx,tsx}',
  ],
  theme: {
    extend: {
      colors: {
        amoled: '#000000',
        background: 'var(--neo-bg)',
        card: 'var(--neo-card)',
        'card-sub': 'var(--neo-card-sub)',
        border: 'var(--neo-border)',
        accent: 'var(--neo-accent)',
        'accent-text': 'var(--neo-accent-text)',
        muted: 'var(--neo-muted)',
        foreground: 'var(--neo-text)',
      },
      fontFamily: {
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', '"Liberation Mono"', '"Courier New"', 'monospace'],
      },
      boxShadow: {
        neobrutal: '4px 4px 0px 0px var(--neo-shadow)',
        'neobrutal-sm': '2px 2px 0px 0px var(--neo-shadow)',
        'neobrutal-lg': '6px 6px 0px 0px var(--neo-shadow)',
      },
    },
  },
  plugins: [],
}


