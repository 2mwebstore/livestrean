/** The shared API client from plugins/01.api.ts. */
export function useApi() {
  return useNuxtApp().$api
}
