export const BACKGROUND_SETTINGS_STORAGE_KEY = 'koyomi-background-settings'

export const ALLOWED_BACKGROUND_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/avif'
] as const

export const MAX_BACKGROUND_SIZE = 20 * 1024 * 1024
