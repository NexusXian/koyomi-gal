import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import {
  ALLOWED_BACKGROUND_TYPES,
  BACKGROUND_PRESETS,
  BACKGROUND_SETTINGS_STORAGE_KEY,
  MAX_BACKGROUND_SIZE
} from '~/constants/backgrounds'
import { useBackgroundStorage } from '~/composables/useBackgroundStorage'
import {
  DEFAULT_BACKGROUND_SETTINGS,
  type BackgroundSettings,
  type BackgroundSize,
  type BackgroundSource
} from '~/types/background'

const PERSISTENCE_DELAY = 250

function createDefaultSettings(): BackgroundSettings {
  return { ...DEFAULT_BACKGROUND_SETTINGS }
}

function isBackgroundSource(value: unknown): value is BackgroundSource {
  return value === 'none' || value === 'preset' || value === 'custom'
}

function normalizeSettings(value: unknown): BackgroundSettings {
  if (!value || typeof value !== 'object') {
    return createDefaultSettings()
  }

  const candidate = value as Partial<BackgroundSettings>
  const source = isBackgroundSource(candidate.source) ? candidate.source : 'none'
  const presetId = typeof candidate.presetId === 'string' ? candidate.presetId : null
  const customImageId =
    typeof candidate.customImageId === 'string' ? candidate.customImageId : null
  const opacity =
    typeof candidate.opacity === 'number' && Number.isFinite(candidate.opacity)
      ? Math.min(1, Math.max(0, candidate.opacity))
      : DEFAULT_BACKGROUND_SETTINGS.opacity
  const blur =
    typeof candidate.blur === 'number' && Number.isFinite(candidate.blur)
      ? Math.min(20, Math.max(0, candidate.blur))
      : DEFAULT_BACKGROUND_SETTINGS.blur
  const position =
    typeof candidate.position === 'string' && candidate.position.trim()
      ? candidate.position
      : DEFAULT_BACKGROUND_SETTINGS.position
  const size: BackgroundSize = candidate.size === 'contain' ? 'contain' : 'cover'

  if (
    source === 'preset' &&
    !BACKGROUND_PRESETS.some((preset) => preset.id === presetId)
  ) {
    return {
      ...createDefaultSettings(),
      opacity,
      blur,
      position,
      size,
      customImageId
    }
  }

  if (source === 'custom' && !customImageId) {
    return {
      ...createDefaultSettings(),
      opacity,
      blur,
      position,
      size,
      presetId
    }
  }

  return {
    source,
    presetId,
    customImageId,
    opacity,
    blur,
    position,
    size
  }
}

export const useBackgroundStore = defineStore('background', () => {
  const settings = ref<BackgroundSettings>(createDefaultSettings())
  const initialized = ref(false)
  const initializing = ref(false)
  const customImageUrl = ref<string | null>(null)
  const storage = useBackgroundStorage()
  let persistenceTimer: ReturnType<typeof setTimeout> | null = null
  let pendingPersistence = false

  const activePreset = computed(() => {
    if (settings.value.source !== 'preset') {
      return null
    }

    return (
      BACKGROUND_PRESETS.find((preset) => preset.id === settings.value.presetId) ?? null
    )
  })

  const backgroundUrl = computed(() => {
    if (settings.value.source === 'preset') {
      return activePreset.value?.src ?? null
    }

    if (settings.value.source === 'custom') {
      return customImageUrl.value
    }

    return null
  })

  const hasBackground = computed(() => Boolean(backgroundUrl.value))

  function persistSettings(): void {
    if (!import.meta.client || !initialized.value) {
      return
    }

    try {
      localStorage.setItem(BACKGROUND_SETTINGS_STORAGE_KEY, JSON.stringify(settings.value))
    } catch {
      // Browsers may block local storage in private or restricted contexts.
    }
  }

  function schedulePersistence(): void {
    if (!import.meta.client) {
      return
    }

    if (!initialized.value) {
      pendingPersistence = true
      return
    }

    if (persistenceTimer) {
      clearTimeout(persistenceTimer)
    }

    persistenceTimer = setTimeout(() => {
      persistSettings()
      persistenceTimer = null
    }, PERSISTENCE_DELAY)
  }

  function setCustomImageUrl(url: string | null): void {
    if (customImageUrl.value && customImageUrl.value !== url) {
      URL.revokeObjectURL(customImageUrl.value)
    }

    customImageUrl.value = url
  }

  async function initialize(): Promise<void> {
    if (!import.meta.client || initialized.value || initializing.value) {
      return
    }

    initializing.value = true
    let shouldPersist = false

    try {
      try {
        const savedSettings = localStorage.getItem(BACKGROUND_SETTINGS_STORAGE_KEY)
        settings.value = savedSettings
          ? normalizeSettings(JSON.parse(savedSettings))
          : createDefaultSettings()
      } catch {
        settings.value = createDefaultSettings()
      }

      if (settings.value.source === 'custom' && settings.value.customImageId) {
        try {
          const blob = await storage.getBackground(settings.value.customImageId)
          if (blob) {
            setCustomImageUrl(URL.createObjectURL(blob))
          } else {
            settings.value.source = 'none'
            settings.value.customImageId = null
            shouldPersist = true
          }
        } catch {
          settings.value.source = 'none'
          settings.value.customImageId = null
          shouldPersist = true
        }
      }
    } finally {
      initialized.value = true
      initializing.value = false
      if (shouldPersist || pendingPersistence) {
        persistSettings()
        pendingPersistence = false
      }
    }
  }

  function selectPreset(id: string): void {
    if (!BACKGROUND_PRESETS.some((preset) => preset.id === id)) {
      return
    }

    settings.value.source = 'preset'
    settings.value.presetId = id
    schedulePersistence()
  }

  async function selectCustomImage(id: string): Promise<void> {
    if (!import.meta.client) {
      return
    }

    const blob = await storage.getBackground(id)
    if (!blob) {
      throw new Error('自定义背景图片已不存在')
    }

    setCustomImageUrl(URL.createObjectURL(blob))
    settings.value.customImageId = id
    settings.value.source = 'custom'
    schedulePersistence()
  }

  async function uploadCustomImage(file: File): Promise<void> {
    if (!ALLOWED_BACKGROUND_TYPES.includes(file.type as (typeof ALLOWED_BACKGROUND_TYPES)[number])) {
      throw new Error('不支持的图片格式')
    }

    if (file.size > MAX_BACKGROUND_SIZE) {
      throw new Error('背景图片不能超过 10MB')
    }

    const id = await storage.saveBackground(file)
    const previousImageId = settings.value.customImageId

    if (previousImageId) {
      try {
        await storage.deleteBackground(previousImageId)
      } catch (error) {
        try {
          await storage.deleteBackground(id)
        } catch {
          // Keep the original image active when replacement cleanup cannot finish.
        }
        throw error
      }
    }

    setCustomImageUrl(URL.createObjectURL(file))
    settings.value.customImageId = id
    settings.value.source = 'custom'
    schedulePersistence()
  }

  function setOpacity(value: number): void {
    settings.value.opacity = Math.min(1, Math.max(0, value))
    schedulePersistence()
  }

  function setBlur(value: number): void {
    settings.value.blur = Math.min(20, Math.max(0, value))
    schedulePersistence()
  }

  function setPosition(value: string): void {
    if (!value.trim()) {
      return
    }

    settings.value.position = value
    schedulePersistence()
  }

  function setSize(value: BackgroundSize): void {
    settings.value.size = value
    schedulePersistence()
  }

  function disableBackground(): void {
    settings.value.source = 'none'
    schedulePersistence()
  }

  async function deleteCustomImage(): Promise<void> {
    const id = settings.value.customImageId
    if (id) {
      await storage.deleteBackground(id)
    }

    setCustomImageUrl(null)
    settings.value.customImageId = null
    if (settings.value.source === 'custom') {
      settings.value.source = 'none'
    }
    schedulePersistence()
  }

  async function reset(): Promise<void> {
    try {
      await storage.clearBackgrounds()
    } catch {
      if (settings.value.customImageId) {
        throw new Error('无法删除已保存的自定义背景')
      }
    }

    setCustomImageUrl(null)
    settings.value = createDefaultSettings()
    schedulePersistence()
  }

  function dispose(): void {
    if (persistenceTimer) {
      clearTimeout(persistenceTimer)
      persistSettings()
      persistenceTimer = null
    }
    setCustomImageUrl(null)
  }

  return {
    settings,
    initialized,
    customImageUrl,
    activePreset,
    backgroundUrl,
    hasBackground,
    initialize,
    selectPreset,
    selectCustomImage,
    uploadCustomImage,
    setOpacity,
    setBlur,
    setPosition,
    setSize,
    disableBackground,
    deleteCustomImage,
    reset,
    setCustomImageUrl,
    dispose
  }
})
