// Load the logged-in user once per page load. For SSR pages this runs on
// the server and the result is handed to the browser with the Pinia state;
// for client-only pages (dashboard, admin) it runs in the browser.
export default defineNuxtPlugin(async () => {
  const auth = useAuthStore()
  if (auth.loaded) return

  if (import.meta.server) {
    const cookie = useRequestHeaders(['cookie']).cookie ?? ''
    if (!cookie.includes(`${useRuntimeConfig().public.sessionCookie}=`)) {
      auth.markGuest()
      return
    }
  }
  await auth.fetchMe()
})
