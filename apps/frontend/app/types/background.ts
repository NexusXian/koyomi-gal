export type BackgroundSource = 'none' | 'preset' | 'custom'

export type BackgroundSize = 'cover' | 'contain'

export interface BackgroundPreset {
  id: string
  name: string
  src: string
  thumbnail?: string
}

export interface BackgroundPresetData {
  id: number
  key: string
  name: string
  image_url: string
  sort_order: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface BackgroundPresetPayload {
  name: string
  image_url: string
  sort_order: number
  is_active: boolean
}

export interface BackgroundSettings {
  source: BackgroundSource
  presetId: string | null
  customImageId: number | null
  opacity: number
  blur: number
  position: string
  size: BackgroundSize
}

export const DEFAULT_BACKGROUND_SETTINGS: BackgroundSettings = {
  source: 'none',
  presetId: null,
  customImageId: null,
  opacity: 0.35,
  blur: 0,
  position: 'center center',
  size: 'cover'
}

export interface UserPreferences {
  background_source: BackgroundSource
  background_asset_id: number | null
  background_preset: string | null
  background_opacity: number
  background_blur: number
  background_position: string
  background_size: BackgroundSize
  background_image_url: string
}

export interface UserPreferencesPayload {
  background_source: BackgroundSource
  background_asset_id?: number | null
  background_preset?: string | null
  background_opacity: number
  background_blur: number
  background_position: string
  background_size: BackgroundSize
}
