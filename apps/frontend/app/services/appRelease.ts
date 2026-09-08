import type { ApiClient, ApiResponse } from '~/types/api'
import type { AppRelease, AppReleasePayload } from '~/types/appRelease'
import type { PaginatedData, PaginationParams } from '~/types/content'
import { unwrapApiData } from '~/utils/api'

type AppReleaseListData = PaginatedData<AppRelease> | AppRelease[]

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

function normalizePage(
  data: AppReleaseListData,
  params: PaginationParams
): PaginatedData<AppRelease> {
  if (Array.isArray(data)) {
    return {
      items: data,
      total: data.length,
      page: params.page ?? 1,
      limit: params.limit ?? data.length
    }
  }

  return {
    items: Array.isArray(data.items) ? data.items : [],
    total: Number.isFinite(data.total) ? data.total : 0,
    page: Number.isFinite(data.page) ? data.page : (params.page ?? 1),
    limit: Number.isFinite(data.limit) ? data.limit : (params.limit ?? 20)
  }
}

export function createAppReleaseService(api: ApiClient) {
  return {
    async listAdmin(
      params: PaginationParams
    ): Promise<PaginatedData<AppRelease>> {
      const data = unwrapApiData(
        await api<ApiResponse<AppReleaseListData>>(
          '/api/v1/admin/app/releases',
          { query: params }
        ),
        '版本列表加载失败'
      )
      return normalizePage(data, params)
    },

    async getAdmin(id: number): Promise<AppRelease> {
      return unwrapApiData(
        await api<ApiResponse<AppRelease>>(`/api/v1/admin/app/releases/${id}`),
        '版本加载失败'
      )
    },

    async create(payload: AppReleasePayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>('/api/v1/admin/app/releases', {
          method: 'POST',
          body: payload
        }),
        '版本创建失败'
      )
    },

    async update(id: number, payload: AppReleasePayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/app/releases/${id}`, {
          method: 'PUT',
          body: payload
        }),
        '版本更新失败'
      )
    },

    async remove(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/app/releases/${id}`, {
          method: 'DELETE'
        }),
        '版本删除失败'
      )
    },

    async publish(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/app/releases/${id}/publish`, {
          method: 'PATCH'
        }),
        '版本发布失败'
      )
    },

    async disable(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/app/releases/${id}/disable`, {
          method: 'PATCH'
        }),
        '版本停用失败'
      )
    }
  }
}
