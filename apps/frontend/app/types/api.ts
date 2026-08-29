import type { FetchOptions, FetchRequest } from 'ofetch'

export interface ApiResponse<T = never> {
  code: number
  data?: T
  msg: string
}

export interface ApiRequestOptions extends FetchOptions {
  skipAuth?: boolean
  skipRefresh?: boolean
}

export interface ApiClient {
  <T = unknown>(
    request: FetchRequest,
    options?: ApiRequestOptions
  ): Promise<T>
}
