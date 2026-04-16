export function resolveRouteRedirect(meta = {}, hasToken = false) {
    if (meta.auth && !hasToken) return '/login'
    if (meta.guest && hasToken) return '/'
    return null
}
