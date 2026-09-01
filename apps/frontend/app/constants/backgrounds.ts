import type { BackgroundPreset } from '~/types/background'

export const BACKGROUND_SETTINGS_STORAGE_KEY = 'koyomi-background-settings'

export const ALLOWED_BACKGROUND_TYPES = [
  'image/jpeg',
  'image/png',
  'image/webp',
  'image/avif'
] as const

export const MAX_BACKGROUND_SIZE = 10 * 1024 * 1024

export const BACKGROUND_PRESETS: BackgroundPreset[] = [
  {
    id: 'default-01',
    name: '暮色花海',
    src: '/images/backgrounds/default-01.webp',
    thumbnail: '/images/backgrounds/default-01-thumb.webp'
  },
  {
    id: 'default-02',
    name: '星夜海岸',
    src: '/images/backgrounds/default-02.webp',
    thumbnail: '/images/backgrounds/default-02-thumb.webp'
  },
  {
    id: 'default-03',
    name: '青空云间',
    src: '/images/backgrounds/default-03.webp',
    thumbnail: '/images/backgrounds/default-03-thumb.webp'
  },
  {
    id: 'default-04',
    name: '樱色晨光',
    src: '/images/backgrounds/default-04.webp',
    thumbnail: '/images/backgrounds/default-04-thumb.webp'
  },
  {
    id: 'default-05',
    name: '静谧森林',
    src: '/images/backgrounds/default-05.webp',
    thumbnail: '/images/backgrounds/default-05-thumb.webp'
  }
]
