import { describe, expect, it } from 'vitest'
import { resolveRouteRedirect } from '../src/utils/authRoute'

describe('auth route guard helper', () => {
    it('redirects anonymous users away from protected pages', () => {
        expect(resolveRouteRedirect({ auth: true }, false)).toBe('/login')
    })

    it('redirects logged-in users away from guest pages', () => {
        expect(resolveRouteRedirect({ guest: true }, true)).toBe('/')
    })

    it('lets normal navigation pass through', () => {
        expect(resolveRouteRedirect({}, false)).toBe(null)
        expect(resolveRouteRedirect({ auth: true }, true)).toBe(null)
        expect(resolveRouteRedirect({ guest: true }, false)).toBe(null)
    })
})
