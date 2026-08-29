import { AUTH_ENDPOINTS } from '~/constants/api'
import type { ApiClient, ApiResponse } from '~/types/api'
import type { AuthSessionPayload, LoginCredentials } from '~/types/auth'
import type { UserSession } from '~/types/user'

function getResponseData<T>(response: ApiResponse<T>): T {
  if (response.code !== 0 || response.data === undefined) {
    throw new Error(response.msg || 'Request failed')
  }

  return response.data
}

function toUserSession(payload: AuthSessionPayload): UserSession {
  return {
    user: payload.user,
    accessToken: payload.token
  }
}

export function createAuthService(api: ApiClient) {
  return {
    async login(credentials: LoginCredentials): Promise<UserSession> {
      const response = await api<ApiResponse<AuthSessionPayload>>(
        AUTH_ENDPOINTS.login,
        {
          method: 'POST',
          body: credentials,
          skipAuth: true,
          skipRefresh: true
        }
      )

      return toUserSession(getResponseData(response))
    },

    async refresh(): Promise<UserSession> {
      const response = await api<ApiResponse<AuthSessionPayload>>(
        AUTH_ENDPOINTS.refresh,
        {
          method: 'POST',
          skipAuth: true,
          skipRefresh: true
        }
      )

      return toUserSession(getResponseData(response))
    },

    async logout(): Promise<void> {
      const response = await api<ApiResponse>(AUTH_ENDPOINTS.logout, {
        method: 'POST',
        skipAuth: true,
        skipRefresh: true
      })

      if (response.code !== 0) {
        throw new Error(response.msg || 'Logout failed')
      }
    }
  }
}
