import type { ApiClient, ApiResponse } from '~/types/api'
import type { SiteChangelog, SiteChangelogPayload } from '~/types/siteChangelog'
import type { PaginatedData, PaginationParams } from '~/types/content'
import { unwrapApiData } from '~/utils/api'

type ChangelogListData = PaginatedData<SiteChangelog> | SiteChangelog[]

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

export function createSiteChangelogService(api: ApiClient) {
  return {
    async list(): Promise<SiteChangelog[]> {
      const data = unwrapApiData(
        await api<ApiResponse<{ items: SiteChangelog[] }>>('/api/v1/changelogs', {
          skipAuth: true,
          skipRefresh: true
        }),
        '更新日志加载失败'
      )
      return Array.isArray(data.items) ? data.items : []
    },

    async listAdmin(
      params: PaginationParams
    ): Promise<PaginatedData<SiteChangelog>> {
      const data = unwrapApiData(
        await api<ApiResponse<ChangelogListData>>(
          '/api/v1/admin/changelogs',
          { query: params }
        ),
        '更新日志列表加载失败'
      )
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
    },

    async getAdmin(id: number): Promise<SiteChangelog> {
      return unwrapApiData(
        await api<ApiResponse<SiteChangelog>>(
          `/api/v1/admin/changelogs/${id}`
        ),
        '更新日志加载失败'
      )
    },

    async create(payload: SiteChangelogPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>('/api/v1/admin/changelogs', {
          method: 'POST',
          body: payload
        }),
        '更新日志创建失败'
      )
    },

    async update(id: number, payload: SiteChangelogPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/changelogs/${id}`, {
          method: 'PUT',
          body: payload
        }),
        '更新日志更新失败'
      )
    },

    async remove(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/changelogs/${id}`, {
          method: 'DELETE'
        }),
        '更新日志删除失败'
      )
    }
  }
}
