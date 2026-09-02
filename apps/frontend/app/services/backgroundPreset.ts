import type { ApiClient, ApiResponse } from '~/types/api'
import type {
  BackgroundPreset,
  BackgroundPresetData,
  BackgroundPresetPayload
} from '~/types/background'
import type { PaginatedData, PaginationParams } from '~/types/content'
import { unwrapApiData } from '~/utils/api'

function toPreset(data: BackgroundPresetData): BackgroundPreset {
  return { id: data.key, name: data.name, src: data.image_url }
}

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

export function createBackgroundPresetService(api: ApiClient) {
  return {
    async listPresets(): Promise<BackgroundPreset[]> {
      const items = unwrapApiData(
        await api<ApiResponse<BackgroundPresetData[]>>(
          '/api/v1/background-presets',
          { skipAuth: true, skipRefresh: true }
        ),
        '背景预设加载失败'
      )
      return items.map(toPreset)
    },

    async listAdminPresets(
      params: PaginationParams
    ): Promise<PaginatedData<BackgroundPresetData>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<BackgroundPresetData>>>(
          '/api/v1/admin/background-presets',
          { query: params }
        ),
        '背景预设列表加载失败'
      )
    },

    async createPreset(payload: BackgroundPresetPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>('/api/v1/admin/background-presets', {
          method: 'POST',
          body: payload
        }),
        '背景预设创建失败'
      )
    },

    async updatePreset(
      id: number,
      payload: BackgroundPresetPayload
    ): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/background-presets/${id}`, {
          method: 'PUT',
          body: payload
        }),
        '背景预设更新失败'
      )
    },

    async deletePreset(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/background-presets/${id}`, {
          method: 'DELETE'
        }),
        '背景预设删除失败'
      )
    }
  }
}
