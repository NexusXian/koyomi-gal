export type HomeDeveloper =
  | string
  | {
      id?: number
      name?: string
    }
  | null

export interface HomeBanner {
  id: number
  title: string
  subtitle?: string | null
  image_url?: string | null
  link_type?: string | null
  link_value?: string | null
}

export interface HomeArticle {
  id: number
  title: string
  summary?: string | null
  cover_url?: string | null
  type: string
  is_pinned: boolean
  published_at?: string | null
}

export interface HomeGalgame {
  id: number
  title: string
  cover_url?: string | null
  developer?: HomeDeveloper
  rating_average?: number | null
  favorite_count: number
  release_date?: string | null
  updated_at?: string | null
}

export interface HomePost {
  id: number
  title: string
  author_id?: number
  author_name?: string | null
  galgame_id?: number | null
  galgame_title?: string | null
  author?: {
    id?: number
    username?: string
    avatar?: string
  } | null
  galgame?: {
    id?: number
    title?: string
    cover_url?: string
  } | null
  like_count: number
  comment_count: number
  favorite_count: number
  created_at: string
}

export interface HomeData {
  banners: HomeBanner[]
  announcements: HomeArticle[]
  latest_galgames: HomeGalgame[]
  popular_galgames: HomeGalgame[]
  latest_posts: HomePost[]
  popular_posts: HomePost[]
}
