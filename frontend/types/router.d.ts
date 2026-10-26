import type { Permission } from './auth'

declare module '#app' {
  interface PageMeta {
    /** Checked by middleware/permission.ts. UX only; the API enforces it too. */
    permission?: Permission
  }
}

export {}
