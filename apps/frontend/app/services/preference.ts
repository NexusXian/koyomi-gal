import { USER_ENDPOINTS } from '~/constants/api'
import type { ApiClient, ApiResponse } from '~/types/api'
import type {
  UserPreferences,
  UserPreferencesPayload
} from '~/types/background'
import { unwrapApiData } from '~/utils/api'

export function createPreferenceService(api: ApiClient) {
  return {
    async getPreferences(): Promise<UserPreferences> {
      return unwrapApiData(
        await api<ApiResponse<UserPreferences>>(USER_ENDPOINTS.mePreferences),
        '背景设置加载失败'
      )
    },

    async updatePreferences(
      payload: UserPreferencesPayload
    ): Promise<UserPreferences> {
      return unwrapApiData(
        await api<ApiResponse<UserPreferences>>(USER_ENDPOINTS.mePreferences, {
          method: 'PATCH',
          body: payload
        }),
        '背景设置保存失败'
      )
    }
  }
}
