import { defineStore } from 'pinia'
import api from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem('token'),
    refreshToken: localStorage.getItem('refreshToken')
  }),

  getters: {
    isAuthenticated: (state) => !!state.token
  },

  actions: {
    async register(username, password) {
      await api.register({ username, password })
      // Registration succeeds → redirect to login (no token returned)
    },

    async login(username, password) {
      const res = await api.login({ username, password })
      const d = res.data.data
      this.token = d.token
      this.refreshToken = d.refreshToken
      localStorage.setItem('token', d.token)
      localStorage.setItem('refreshToken', d.refreshToken)
    },

    logout() {
      this.token = null
      this.refreshToken = null
      localStorage.removeItem('token')
      localStorage.removeItem('refreshToken')
    }
  }
})
