export const API_PREFIX = '/api/v1'

export const AUTH_ENDPOINTS = {
  register: `${API_PREFIX}/auth/register`,
  login: `${API_PREFIX}/auth/login`,
  refresh: `${API_PREFIX}/auth/refresh`,
  logout: `${API_PREFIX}/auth/logout`,
  verificationCodes: `${API_PREFIX}/auth/verification-codes`
} as const

export const USER_ENDPOINTS = {
  me: `${API_PREFIX}/me`,
  mePreferences: `${API_PREFIX}/me/preferences`
} as const
