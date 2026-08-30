import { storeToRefs } from 'pinia'
import { createAuthService } from '~/services/auth'
import type { LoginCredentials, RegistrationCredentials } from '~/types/auth'
import type { User, UserSession } from '~/types/user'
import { useUserStore } from '~/stores/user'

export function useAuth() {
  const { $api, $refreshSession, $invalidateRefreshSession } = useNuxtApp()
  const userStore = useUserStore()
  const authService = createAuthService($api)
  const { user, status, initialized, isAuthenticated } =
    storeToRefs(userStore)

  const login = async (credentials: LoginCredentials): Promise<User> => {
    $invalidateRefreshSession()
    userStore.setStatus('loading')

    try {
      const session = await authService.login(credentials)
      userStore.setSession(session)
      return session.user
    } catch (error: unknown) {
      userStore.clearSession()
      throw error
    }
  }

  const register = (credentials: RegistrationCredentials): Promise<string> =>
    authService.register(credentials)

  const sendRegistrationCode = (email: string): Promise<string> =>
    authService.sendVerificationCode(email, 'register')

  const refresh = (): Promise<UserSession> => $refreshSession()

  const initialize = async (): Promise<User | null> => {
    if (userStore.getInitialized) {
      return userStore.getUser
    }

    if (import.meta.server) {
      return null
    }

    try {
      const session = await refresh()
      return session.user
    } catch {
      return null
    }
  }

  const logout = async (): Promise<void> => {
    $invalidateRefreshSession()
    userStore.clearSession()

    await authService.logout()
  }

  return {
    user,
    status,
    initialized,
    isAuthenticated,
    register,
    sendRegistrationCode,
    login,
    refresh,
    initialize,
    logout
  }
}
