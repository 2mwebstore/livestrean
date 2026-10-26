// Checks definePageMeta({ permission }). Run after `auth`.
// UX only: the API rejects the same requests on its own.
export default defineNuxtRouteMiddleware((to) => {
  const permission = to.meta.permission
  if (permission && !useAuthStore().can(permission)) {
    return abortNavigation(createError({ statusCode: 403, statusMessage: 'You don\'t have access to this page' }))
  }
})
