import type { ApiClient, ApiResponse } from '~/types/api'
import type { FeedbackData, FeedbackPayload, FeedbackType } from '~/types/feedback'
import type { PaginatedData } from '~/types/content'
import { unwrapApiData } from '~/utils/api'

export interface AdminFeedbackParams {
  page?: number
  limit?: number
  type?: FeedbackType
  handled?: boolean
}

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

export function createFeedbackService(api: ApiClient) {
  return {
    async submitFeedback(payload: FeedbackPayload): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>('/api/v1/feedback', {
          method: 'POST',
          body: payload,
          skipAuth: true,
          skipRefresh: true
        }),
        '反馈提交失败'
      )
    },

    async listAdminFeedback(
      params: AdminFeedbackParams
    ): Promise<PaginatedData<FeedbackData>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<FeedbackData>>>(
          '/api/v1/admin/feedback',
          { query: params }
        ),
        '反馈列表加载失败'
      )
    },

    async handleFeedback(id: number, handled: boolean): Promise<void> {
      ensureSuccess(
        await api<ApiResponse>(`/api/v1/admin/feedback/${id}/handle`, {
          method: 'PUT',
          body: { handled }
        }),
        '反馈处理失败'
      )
    }
  }
}
