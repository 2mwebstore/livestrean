import { defineConfig } from 'vitest/config'

// Unit tests for framework-free code (utils). End-to-end browser tests
// live in tests/e2e and run against the full Docker stack.
export default defineConfig({
  test: {
    include: ['tests/unit/**/*.test.ts'],
  },
})
