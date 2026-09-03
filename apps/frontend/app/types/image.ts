export type ImageCategory =
  | 'avatars'
  | 'posts'
  | 'comments'
  | 'galgames'
  | 'backgrounds'
  | 'profile-banners'
  | 'banners'
  | 'admin'

export const IMAGE_STATUSES = ['pending', 'active', 'deleted', 'failed'] as const
export type ImageStatus = (typeof IMAGE_STATUSES)[number]

export interface PresignImageRequest {
  filename: string
  content_type: string
  size: number
  category: ImageCategory
}

export interface PresignImageData {
  id: number
  object_key: string
  upload_url: string
  expires_in: number
}

export interface ImageAsset {
  id: number
  url: string
  object_key: string
  original_name?: string | null
  mime_type: string
  extension?: string | null
  size: number
  width?: number | null
  height?: number | null
  category: ImageCategory
  status: number
  created_at?: string | null
  deleted_at?: string | null
  user_id?: number | null
}

export interface AdminImageQuery {
  page?: number
  limit?: number
  category?: ImageCategory
  user_id?: number
  status?: number
}
