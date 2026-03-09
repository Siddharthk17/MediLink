import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./vitest.setup.ts'],
    include: ['src/**/*.test.{ts,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html'],
      thresholds: { statements: 80, branches: 75, functions: 80, lines: 80 },
      include: ['src/**/*.{ts,tsx}'],
      exclude: [
        'node_modules/',
        '.next/',
        'src/app/layout.tsx',
        'src/app/providers.tsx',
        'src/app/globals.css',
        'src/lib/msw/**',
        'src/**/__mocks__/**',
        'src/**/__tests__/**',
        'src/**/types/**',
        'src/app/**/page.tsx',
        'src/app/**/layout.tsx',
        'src/middleware.ts',
        'src/components/aura/**',
      ],
    },
  },
  resolve: {
    alias: { '@': path.resolve(__dirname, './src') },
  },
})
