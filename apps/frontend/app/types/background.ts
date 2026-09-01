export type BackgroundSource = 'none' | 'preset' | 'custom'

export type BackgroundSize = 'cover' | 'contain'

export interface BackgroundPreset {
  id: string
  name: string
  src: string
  thumbnail?: string
}

export interface BackgroundSettings {
  source: BackgroundSource
  presetId: string | null
  customImageId: string | null
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
