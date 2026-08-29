export interface User {
  id: number
  username: string
  email: string
  avatar: string
}

export type UserStatus =
  | 'idle'
  | 'loading'
  | 'authenticated'
  | 'anonymous'

export interface UserSession {
  user: User
  accessToken: string
}
