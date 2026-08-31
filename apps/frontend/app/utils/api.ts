interface ApiEnvelope<T> {
  code?: number | null
  data?: T
  msg?: string
}

export function unwrapApiData<T>(
  response: ApiEnvelope<T> | undefined | null,
  fallback = '请求失败'
): T {
  if (!response || (response.code ?? 0) !== 0 || response.data === undefined) {
    throw new Error(response?.msg || fallback)
  }

  return response.data
}

export function getApiErrorMessage(
  error: unknown,
  fallback = '请求失败，请稍后重试'
): string {
  const candidate = error as {
    data?: { msg?: unknown }
    message?: unknown
    name?: unknown
  }

  if (
    typeof candidate?.data?.msg === 'string' &&
    candidate.data.msg.trim()
  ) {
    return candidate.data.msg
  }

  if (
    error instanceof Error &&
    error.name !== 'FetchError' &&
    error.message &&
    error.message !== 'Request failed'
  ) {
    return error.message
  }

  return fallback
}
