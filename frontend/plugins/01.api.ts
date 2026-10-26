// The one API client. Use it through services/*.service.ts.
export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig()

  // During SSR, call the API directly and pass along the visitor's cookie
  // and IP, so the API sees the same user and rate-limits the right address.
  const headers = import.meta.server
    ? useRequestHeaders(['cookie', 'x-forwarded-for', 'user-agent'])
    : undefined

  const api = $fetch.create({
    baseURL: import.meta.server ? config.apiInternalBase : config.public.apiBase,
    headers,
    timeout: 15_000,
    retry: 0,
    onResponseError({ response }) {
      // Session expired or revoked: forget the user so the UI updates.
      if (response.status === 401 && import.meta.client) {
        useAuthStore().clear()
      }
    },
  })

  return { provide: { api } }
})
