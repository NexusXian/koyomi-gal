import type { HomeArticle, HomeBanner } from '~/types/home'

export interface PaginatedData<T> {
  items: T[]
  total: number
  page: number
  limit: number
}

export interface Article extends HomeArticle {
  content?: string | null
  view_count?: number
  created_at?: string | null
  updated_at?: string | null
  is_published?: boolean
}

export interface BannerAdmin extends HomeBanner {
  sort_order: number
  is_active: boolean
  start_at?: string | null
  end_at?: string | null
  created_at?: string | null
  updated_at?: string | null
}

export interface BannerPayload {
  title: string
  subtitle?: string | null
  image_url: string
  link_type?: string | null
  link_value?: string | null
  sort_order: number
  is_active: boolean
  start_at?: string | null
  end_at?: string | null
}

export interface ArticlePayload {
  title: string
  summary?: string | null
  content: string
  cover_url?: string | null
  type: string
  is_pinned: boolean
  is_published: boolean
  published_at?: string | null
}

export interface ListArticlesParams {
  type?: string
  page?: number
  limit?: number
}

export interface PaginationParams {
  page?: number
  limit?: number
}
