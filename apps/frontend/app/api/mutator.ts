import type { ApiRequestOptions } from '~/types/api'

export function apiMutator<T>(
  url: string,
  options: ApiRequestOptions
): Promise<T> {
  const { $api } = useNuxtApp()
  return $api<T>(url, options)
}
