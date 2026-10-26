// Pages that need a login. The API enforces this independently.
export default defineNuxtRouteMiddleware((to) => {
  if (!useAuthStore().isAuthenticated) {
    return navigateTo({ path: '/login', query: { redirect: to.fullPath } })
  }
})
