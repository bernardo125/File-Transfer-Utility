import { defineConfig } from 'vitest/config';

// Unit tests for pure logic only (no DOM). User journeys are covered by tests/ui (Playwright).
export default defineConfig({ test: { include: ['src/**/*.test.ts'], environment: 'node' } });
