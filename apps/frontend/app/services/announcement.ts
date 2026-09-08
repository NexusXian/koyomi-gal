import type { ApiClient, ApiResponse } from '~/types/api'
import type { Announcement, AnnouncementPayload } from '~/types/announcement'
import type { PaginatedData, PaginationParams } from '~/types/content'
import { unwrapApiData } from '~/utils/api'

type AnnouncementListData = PaginatedData<Announcement> | Announcement[]

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

function normalizePage(
  data: AnnouncementListData,
  params: PaginationParams
): PaginatedData<Announcement> {
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

export function createAnnouncementService(api: ApiClient) {
  return {
    async listActive(platform: 'web' | 'android' | 'ios'): Promise<Announcement[]> {
      return unwrapApiData(
        await api<ApiResponse<Announcement[]>>('/api/v1/announcements/active', {
          query: { platform },
          skipAuth: true,
          skipRefresh: true
        }),
        '公告加载失败'
      )
    },

    async listAdmin(
      params: PaginationParams
    ): Promise<PaginatedData<Announcement>> {
      const data = unwrapApiData(
        await api<ApiResponse<AnnouncementListData>>(
          '/api/v1/admin/announcements',
          { query: params }
        ),
        '公告列表加载失败'
      )
      return normalizePage(data, params)
    },

    async getAdmin(id: number): Promise<Announcement> {
      return unwrapApiData(
        await api<ApiResponse<Announcement>>(
          `/api/v1/admin/announcements/${id}`
        ),
        '公告加载失败'
      )
    },

    async create(payload: AnnouncementPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>('/api/v1/admin/announcements', {
          method: 'POST',
          body: payload
        }),
        '公告创建失败'
      )
    },

    async update(id: number, payload: AnnouncementPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/announcements/${id}`, {
          method: 'PUT',
          body: payload
        }),
        '公告更新失败'
      )
    },

    async remove(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/announcements/${id}`, {
          method: 'DELETE'
        }),
        '公告删除失败'
      )
    },

    async publish(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/announcements/${id}/publish`, {
          method: 'PATCH'
        }),
        '公告发布失败'
      )
    },

    async withdraw(id: number): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/announcements/${id}/withdraw`, {
          method: 'PATCH'
        }),
        '公告撤回失败'
      )
    }
  }
}
