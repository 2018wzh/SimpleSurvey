import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAuthStore } from '../src/stores/auth'

const apiMocks = vi.hoisted(() => ({
    login: vi.fn(),
    register: vi.fn()
}))

vi.mock('../src/api', () => ({
    default: apiMocks
}))

describe('auth store', () => {
    beforeEach(() => {
        setActivePinia(createPinia())
        localStorage.clear()
        apiMocks.login.mockReset()
        apiMocks.register.mockReset()
    })

    it('stores tokens on login and exposes the authenticated state', async () => {
        apiMocks.login.mockResolvedValue({
            data: {
                data: {
                    token: 'token-123',
                    refreshToken: 'refresh-123'
                }
            }
        })

        const auth = useAuthStore()
        await auth.login('alice', 'secret')

        expect(apiMocks.login).toHaveBeenCalledWith({ username: 'alice', password: 'secret' })
        expect(auth.token).toBe('token-123')
        expect(auth.refreshToken).toBe('refresh-123')
        expect(localStorage.getItem('token')).toBe('token-123')
        expect(localStorage.getItem('refreshToken')).toBe('refresh-123')
        expect(auth.isAuthenticated).toBe(true)
    })

    it('calls register without storing credentials and logout clears local storage', async () => {
        apiMocks.register.mockResolvedValue({ data: { data: null } })

        const auth = useAuthStore()
        await auth.register('bob', 'secret')
        auth.logout()

        expect(apiMocks.register).toHaveBeenCalledWith({ username: 'bob', password: 'secret' })
        expect(auth.token).toBe(null)
        expect(auth.refreshToken).toBe(null)
        expect(localStorage.getItem('token')).toBe(null)
        expect(localStorage.getItem('refreshToken')).toBe(null)
        expect(auth.isAuthenticated).toBe(false)
    })
})
