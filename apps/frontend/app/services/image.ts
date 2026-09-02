import type { ApiClient, ApiResponse } from '~/types/api'
import type { PaginatedData } from '~/types/content'
import type {
  AdminImageQuery,
  ImageAsset,
  PresignImageData,
  PresignImageRequest
} from '~/types/image'
import { unwrapApiData } from '~/utils/api'

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

export function createImageService(api: ApiClient) {
  return {
    async presign(payload: PresignImageRequest): Promise<PresignImageData> {
      return unwrapApiData(
        await api<ApiResponse<PresignImageData>>('/api/v1/images/presign', {
          method: 'POST',
          body: payload
        }),
        '上传凭证获取失败'
      )
    },

    async completeUpload(
      id: number,
      payload: { width?: number; height?: number } = {}
    ): Promise<ImageAsset> {
      return unwrapApiData(
        await api<ApiResponse<ImageAsset>>(`/api/v1/images/${id}/complete`, {
          method: 'POST',
          body: payload
        }),
        '图片上传确认失败'
      )
    },

    async getImage(id: number): Promise<ImageAsset> {
      return unwrapApiData(
        await api<ApiResponse<ImageAsset>>(`/api/v1/images/${id}`, {
          skipAuth: true,
          skipRefresh: true
        }),
        '图片信息加载失败'
      )
    },

    async deleteImage(id: number): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/images/${id}`, {
        method: 'DELETE'
      })
      ensureSuccess(response, '图片删除失败')
    },

    async listAdminImages(
      params: AdminImageQuery
    ): Promise<PaginatedData<ImageAsset>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<ImageAsset>>>(
          '/api/v1/admin/images',
          { query: params }
        ),
        '图片列表加载失败'
      )
    },

    async deleteAdminImage(id: number): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/admin/images/${id}`, {
        method: 'DELETE'
      })
      ensureSuccess(response, '图片删除失败')
    }
  }
}
