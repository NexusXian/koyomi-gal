import type { FetchError, FetchRequest } from 'ofetch'
import { createAuthService } from '~/services/auth'
import type { ApiClient, ApiRequestOptions } from '~/types/api'
import type { UserSession } from '~/types/user'
import { useUserStore } from '~/stores/user'

function getErrorStatus(error: unknown): number | undefined {
  const fetchError = error as FetchError
  return fetchError.statusCode ?? fetchError.status ?? fetchError.response?.status
}

export default defineNuxtPlugin(() => {
  const config = useRuntimeConfig()
  const userStore = useUserStore()
  let refreshPromise: Promise<UserSession> | null = null
  let refreshGeneration = 0
  let refreshBlocked = false

  const rawApi = $fetch.create({
    baseURL: config.public.apiBase,
    credentials: 'include',
    onRequest({ options }) {
      const apiOptions = options as ApiRequestOptions
      const headers = new Headers(options.headers)

      if (
        !apiOptions.skipAuth &&
        userStore.getAccessToken &&
        !headers.has('Authorization')
      ) {
        headers.set('Authorization', `Bearer ${userStore.getAccessToken}`)
      }

      options.headers = headers
    }
  })

  const rawClient = rawApi as ApiClient
  const rawAuthService = createAuthService(rawClient)

  const invalidateRefreshSession = (): void => {
    refreshGeneration += 1
    refreshPromise = null
    refreshBlocked = true
  }

  const refreshSession = async (): Promise<UserSession> => {
    if (import.meta.server) {
      throw new Error('Token refresh is only available on the client')
    }

    if (!refreshPromise) {
      const generation = refreshGeneration
      userStore.setStatus('loading')
      refreshPromise = rawAuthService
        .refresh()
        .then((session) => {
          if (generation !== refreshGeneration) {
            throw new Error('Token refresh was invalidated')
          }

          refreshBlocked = false
          userStore.setSession(session)
          return session
        })
        .catch((error: unknown) => {
          if (generation === refreshGeneration) {
            refreshBlocked = true
            userStore.clearSession()
          }

          throw error
        })
        .finally(() => {
          if (generation === refreshGeneration) {
            refreshPromise = null
          }
        })
    }

    return refreshPromise
  }

  const api: ApiClient = async <T = unknown>(
    request: FetchRequest,
    options: ApiRequestOptions = {}
  ): Promise<T> => {
    try {
      return await rawClient<T>(request, options)
    } catch (error: unknown) {
      if (
        getErrorStatus(error) !== 401 ||
        options.skipRefresh ||
        import.meta.server
      ) {
        throw error
      }

      if (refreshBlocked) {
        if (!userStore.getAccessToken) {
          throw error
        }

        refreshBlocked = false
      }

      await refreshSession()

      return rawClient<T>(request, {
        ...options,
        skipRefresh: true
      })
    }
  }

  return {
    provide: {
      api,
      refreshSession,
      invalidateRefreshSession
    }
  }
})

declare module '#app' {
  interface NuxtApp {
    $api: ApiClient
    $refreshSession: () => Promise<UserSession>
    $invalidateRefreshSession: () => void
  }
}

declare module 'vue' {
  interface ComponentCustomProperties {
    $api: ApiClient
    $refreshSession: () => Promise<UserSession>
    $invalidateRefreshSession: () => void
  }
}
