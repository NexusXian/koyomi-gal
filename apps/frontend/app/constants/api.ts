export const API_PREFIX = '/api/v1'

export const AUTH_ENDPOINTS = {
  login: `${API_PREFIX}/auth/login`,
  refresh: `${API_PREFIX}/auth/refresh`,
  logout: `${API_PREFIX}/auth/logout`
} as const
