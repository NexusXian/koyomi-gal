import type { ApiClient, ApiResponse } from '~/types/api'
import type { HomeData } from '~/types/home'

export function createHomeService(api: ApiClient) {
  return {
    async getHome(): Promise<HomeData> {
      return unwrapApiData(
        await api<ApiResponse<HomeData>>('/api/v1/home', {
          skipAuth: true,
          skipRefresh: true
        }),
        '首页内容加载失败'
      )
    }
  }
}
