import type { User } from '~/types/user'

export interface LoginCredentials {
  email: string
  password: string
}

export interface AuthSessionPayload {
  user: User
  token: string
}
