export type UserRole = 'programmer' | 'safety_reviewer' | 'admin'

export interface AuthUser {
  id: number
  username: string
  display_name: string
  role: UserRole
}

export interface LoginResponse {
  access_token: string
  token_type: 'Bearer'
  expires_in_seconds: number
  user: AuthUser
}
