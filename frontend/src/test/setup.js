import { beforeEach, vi } from 'vitest'

vi.stubGlobal('alert', vi.fn())
vi.stubGlobal('confirm', vi.fn(() => true))
vi.stubGlobal('prompt', vi.fn())

if (!globalThis.crypto) {
    vi.stubGlobal('crypto', {})
}

if (typeof globalThis.crypto.randomUUID !== 'function') {
    let uuidCounter = 0
    globalThis.crypto.randomUUID = vi.fn(() => `uuid-${++uuidCounter}`)
}

function createStorage() {
    const store = new Map()

    return {
        getItem(key) {
            return store.has(key) ? store.get(key) : null
        },
        setItem(key, value) {
            store.set(key, String(value))
        },
        removeItem(key) {
            store.delete(key)
        },
        clear() {
            store.clear()
        }
    }
}

vi.stubGlobal('localStorage', createStorage())
vi.stubGlobal('sessionStorage', createStorage())

if (!navigator.clipboard) {
    Object.defineProperty(navigator, 'clipboard', {
        value: {
            writeText: vi.fn()
        },
        configurable: true
    })
}

beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    sessionStorage.clear()
})
