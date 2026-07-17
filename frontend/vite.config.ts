/// <reference types="vitest" />

import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig(({mode}) => ({
  plugins: [react({fastRefresh: mode !== 'test'})],
  test: {
    environment: 'jsdom',
    setupFiles: './src/test/setup.ts',
  },
}))
