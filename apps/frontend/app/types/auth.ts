import type { User } from '~/types/user'

export interface LoginCredentials {
  account: string
  password: string
}

export interface RegistrationCredentials {
  username: string
  email: string
  password: string
  confirmPassword: string
  verificationCode: string
}

export type VerificationPurpose = 'register' | 'password_reset'

export interface AuthSessionPayload {
  user: User
  token: string
}
