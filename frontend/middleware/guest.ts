// Login and sign-up pages: send logged-in users onward.
export default defineNuxtRouteMiddleware((to) => {
  if (useAuthStore().isAuthenticated) {
    return navigateTo(safeRedirect(to.query.redirect))
  }
})
