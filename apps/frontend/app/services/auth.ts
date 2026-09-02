import {
  AUTH_ENDPOINTS,
  USER_ENDPOINTS
} from '~/constants/api'
import type { ApiClient, ApiResponse } from '~/types/api'
import type {
  AuthSessionPayload,
  LoginCredentials,
  RegistrationCredentials,
  VerificationPurpose
} from '~/types/auth'
import type { User, UserSession } from '~/types/user'

function getResponseData<T>(response: ApiResponse<T>): T {
  if (response.code !== 0 || response.data === undefined) {
    throw new Error(response.msg || 'Request failed')
  }

  return response.data
}

function getResponseMessage(response: ApiResponse, fallback: string): string {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }

  return response.msg || fallback
}

function toUserSession(payload: AuthSessionPayload): UserSession {
  return {
    user: payload.user,
    accessToken: payload.token
  }
}

export function createAuthService(api: ApiClient) {
  return {
    async register(credentials: RegistrationCredentials): Promise<string> {
      const response = await api<ApiResponse>(AUTH_ENDPOINTS.register, {
        method: 'POST',
        body: {
          username: credentials.username,
          email: credentials.email,
          password: credentials.password,
          confirm_password: credentials.confirmPassword,
          verification_code: credentials.verificationCode
        },
        skipAuth: true,
        skipRefresh: true
      })

      return getResponseMessage(response, '注册成功')
    },

    async sendVerificationCode(
      email: string,
      purpose: VerificationPurpose
    ): Promise<string> {
      const response = await api<ApiResponse>(
        AUTH_ENDPOINTS.verificationCodes,
        {
          method: 'POST',
          body: { email, purpose },
          skipAuth: true,
          skipRefresh: true
        }
      )

      return getResponseMessage(response, '验证码已发送')
    },

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
    },

    async updateMe(payload: {
      avatar_asset_id: number | null
    }): Promise<User> {
      const response = await api<ApiResponse<User>>(USER_ENDPOINTS.me, {
        method: 'PATCH',
        body: payload
      })

      return getResponseData(response)
    }
  }
}
