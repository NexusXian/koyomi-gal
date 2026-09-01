import type { ApiClient, ApiResponse } from '~/types/api'
import type {
  Article,
  ArticlePayload,
  BannerAdmin,
  BannerPayload,
  ListArticlesParams,
  PaginatedData,
  PaginationParams
} from '~/types/content'

function ensureSuccess(response: ApiResponse, fallback: string): void {
  if (response.code !== 0) {
    throw new Error(response.msg || fallback)
  }
}

export function createContentService(api: ApiClient) {
  return {
    async listArticles(
      params: ListArticlesParams
    ): Promise<PaginatedData<Article>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<Article>>>('/api/v1/articles', {
          query: params,
          skipAuth: true,
          skipRefresh: true
        }),
        '文章列表加载失败'
      )
    },

    async getArticle(id: number): Promise<Article> {
      return unwrapApiData(
        await api<ApiResponse<Article>>(`/api/v1/articles/${id}`, {
          skipAuth: true,
          skipRefresh: true
        }),
        '文章加载失败'
      )
    },

    async listAdminBanners(
      params: PaginationParams
    ): Promise<PaginatedData<BannerAdmin>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<BannerAdmin>>>(
          '/api/v1/admin/banners',
          { query: params }
        ),
        '轮播图列表加载失败'
      )
    },

    async createBanner(payload: BannerPayload): Promise<void> {
      const response = await api<ApiResponse>('/api/v1/admin/banners', {
        method: 'POST',
        body: payload
      })
      ensureSuccess(response, '轮播图创建失败')
    },

    async updateBanner(id: number, payload: BannerPayload): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/admin/banners/${id}`, {
        method: 'PUT',
        body: payload
      })
      ensureSuccess(response, '轮播图更新失败')
    },

    async deleteBanner(id: number): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/admin/banners/${id}`, {
        method: 'DELETE'
      })
      ensureSuccess(response, '轮播图删除失败')
    },

    async listAdminArticles(
      params: PaginationParams
    ): Promise<PaginatedData<Article>> {
      return unwrapApiData(
        await api<ApiResponse<PaginatedData<Article>>>(
          '/api/v1/admin/articles',
          { query: params }
        ),
        '文章列表加载失败'
      )
    },

    async createArticle(payload: ArticlePayload): Promise<void> {
      const response = await api<ApiResponse>('/api/v1/admin/articles', {
        method: 'POST',
        body: payload
      })
      ensureSuccess(response, '文章创建失败')
    },

    async updateArticle(id: number, payload: ArticlePayload): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/admin/articles/${id}`, {
        method: 'PUT',
        body: payload
      })
      ensureSuccess(response, '文章更新失败')
    },

    async deleteArticle(id: number): Promise<void> {
      const response = await api<ApiResponse>(`/api/v1/admin/articles/${id}`, {
        method: 'DELETE'
      })
      ensureSuccess(response, '文章删除失败')
    }
  }
}
