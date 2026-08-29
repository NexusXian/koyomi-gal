import { defineStore } from 'pinia'
import type { User, UserSession, UserStatus } from '~/types/user'

interface UserState {
  user: User | null
  accessToken: string | null
  status: UserStatus
  initialized: boolean
}

export const useUserStore = defineStore('user', {
  state: (): UserState => ({
    user: null,
    accessToken: null,
    status: 'idle',
    initialized: false
  }),

  getters: {
    getUser: (state): User | null => state.user,
    getAccessToken: (state): string | null => state.accessToken,
    getStatus: (state): UserStatus => state.status,
    getInitialized: (state): boolean => state.initialized,
    isAuthenticated: (state): boolean =>
      Boolean(state.user && state.accessToken)
  },

  actions: {
    setUser(user: User | null): void {
      this.user = user
    },

    setAccessToken(accessToken: string | null): void {
      this.accessToken = accessToken
    },

    setStatus(status: UserStatus): void {
      this.status = status
    },

    setInitialized(initialized: boolean): void {
      this.initialized = initialized
    },

    setSession({ user, accessToken }: UserSession): void {
      this.user = user
      this.accessToken = accessToken
      this.status = 'authenticated'
      this.initialized = true
    },

    clearSession(): void {
      this.user = null
      this.accessToken = null
      this.status = 'anonymous'
      this.initialized = true
    }
  }
})
