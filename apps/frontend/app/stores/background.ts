import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import {
  BACKGROUND_SETTINGS_STORAGE_KEY
} from '~/constants/backgrounds'
import { useImageUpload } from '~/composables/useImageUpload'
import { createBackgroundPresetService } from '~/services/backgroundPreset'
import { createPreferenceService } from '~/services/preference'
import { useUserStore } from '~/stores/user'
import {
  DEFAULT_BACKGROUND_SETTINGS,
  type BackgroundPreset,
  type BackgroundSettings,
  type BackgroundSize,
  type BackgroundSource,
  type UserPreferences
} from '~/types/background'

const PERSISTENCE_DELAY = 250

function createDefaultSettings(): BackgroundSettings {
  return { ...DEFAULT_BACKGROUND_SETTINGS }
}

function isBackgroundSource(value: unknown): value is BackgroundSource {
  return value === 'none' || value === 'preset' || value === 'custom'
}

function clampOpacity(value: number): number {
  return typeof value === 'number' && Number.isFinite(value)
    ? Math.min(1, Math.max(0, value))
    : DEFAULT_BACKGROUND_SETTINGS.opacity
}

function clampBlur(value: number): number {
  return typeof value === 'number' && Number.isFinite(value)
    ? Math.min(20, Math.max(0, value))
    : DEFAULT_BACKGROUND_SETTINGS.blur
}

function normalizeSettings(value: unknown): BackgroundSettings {
  if (!value || typeof value !== 'object') {
    return createDefaultSettings()
  }

  const candidate = value as Partial<BackgroundSettings>
  const source = isBackgroundSource(candidate.source) ? candidate.source : 'none'
  const presetId = typeof candidate.presetId === 'string' ? candidate.presetId : null
  const customImageId =
    typeof candidate.customImageId === 'number' ? candidate.customImageId : null
  const opacity = clampOpacity(candidate.opacity ?? DEFAULT_BACKGROUND_SETTINGS.opacity)
  const blur = clampBlur(candidate.blur ?? DEFAULT_BACKGROUND_SETTINGS.blur)
  const position =
    typeof candidate.position === 'string' && candidate.position.trim()
      ? candidate.position
      : DEFAULT_BACKGROUND_SETTINGS.position
  const size: BackgroundSize = candidate.size === 'contain' ? 'contain' : 'cover'

  // Custom backgrounds live in R2 and require a server asset; local storage
  // can no longer back them, so legacy custom entries fall back to none.
  if (source === 'custom' || customImageId !== null) {
    return { ...createDefaultSettings(), opacity, blur, position, size }
  }

  return { source, presetId, customImageId: null, opacity, blur, position, size }
}

export const useBackgroundStore = defineStore('background', () => {
  const userStore = useUserStore()
  const settings = ref<BackgroundSettings>(createDefaultSettings())
  const presets = ref<BackgroundPreset[]>([])
  const initialized = ref(false)
  const initializing = ref(false)
  const customImageUrl = ref<string | null>(null)
  // True once preferences have been loaded from the server for this login.
  const serverSynced = ref(false)
  let persistenceTimer: ReturnType<typeof setTimeout> | null = null
  let pendingPersistence = false

  const activePreset = computed(() => {
    if (settings.value.source !== 'preset') {
      return null
    }

    return presets.value.find((preset) => preset.id === settings.value.presetId) ?? null
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

  function preferenceService() {
    return createPreferenceService(useNuxtApp().$api)
  }

  function backgroundPresetService() {
    return createBackgroundPresetService(useNuxtApp().$api)
  }

  async function loadPresets(): Promise<void> {
    try {
      presets.value = await backgroundPresetService().listPresets()
    } catch {
      // Keep the current presets when the server list is unavailable.
    }
  }

  function persistSettings(): void {
    if (!import.meta.client || !initialized.value) {
      return
    }

    if (serverSynced.value) {
      const payload = toPayload()
      if (payload.background_source === 'custom' && !payload.background_asset_id) {
        return
      }
      preferenceService().updatePreferences(payload).catch(() => {
        // Preference sync is best-effort; the local state stays applied.
      })
      return
    }

    try {
      localStorage.setItem(
        BACKGROUND_SETTINGS_STORAGE_KEY,
        JSON.stringify(settings.value)
      )
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

  function toPayload() {
    const source = settings.value.source
    return {
      background_source: source,
      background_asset_id:
        source === 'custom' ? settings.value.customImageId : null,
      background_preset: source === 'preset' ? settings.value.presetId : null,
      background_opacity: settings.value.opacity,
      background_blur: settings.value.blur,
      background_position: settings.value.position,
      background_size: settings.value.size
    }
  }

  function applyPreferences(preferences: UserPreferences): void {
    const source = isBackgroundSource(preferences.background_source)
      ? preferences.background_source
      : 'none'
    const imageUrl = preferences.background_image_url || null

    // A custom reference without a resolvable URL (e.g. deleted by an admin)
    // degrades to no background instead of showing a broken one.
    if (source === 'custom' && !imageUrl) {
      settings.value = {
        ...createDefaultSettings(),
        opacity: clampOpacity(preferences.background_opacity),
        blur: clampBlur(preferences.background_blur),
        position: preferences.background_position || DEFAULT_BACKGROUND_SETTINGS.position,
        size: preferences.background_size === 'contain' ? 'contain' : 'cover'
      }
      customImageUrl.value = null
      return
    }

    settings.value = {
      source,
      presetId: preferences.background_preset ?? null,
      customImageId: preferences.background_asset_id ?? null,
      opacity: clampOpacity(preferences.background_opacity),
      blur: clampBlur(preferences.background_blur),
      position: preferences.background_position || DEFAULT_BACKGROUND_SETTINGS.position,
      size: preferences.background_size === 'contain' ? 'contain' : 'cover'
    }
    customImageUrl.value = imageUrl
  }

  function loadLocalSettings(): void {
    try {
      const savedSettings = localStorage.getItem(BACKGROUND_SETTINGS_STORAGE_KEY)
      settings.value = savedSettings
        ? normalizeSettings(JSON.parse(savedSettings))
        : createDefaultSettings()
    } catch {
      settings.value = createDefaultSettings()
    }
  }

  async function loadFromServer(): Promise<void> {
    try {
      const preferences = await preferenceService().getPreferences()
      applyPreferences(preferences)
      serverSynced.value = true
      initialized.value = true
    } catch {
      // Keep the local (guest) settings when preferences cannot be loaded.
    }
  }

  function revertToGuest(): void {
    serverSynced.value = false
    customImageUrl.value = null
    loadLocalSettings()
  }

  async function initialize(): Promise<void> {
    if (!import.meta.client || initialized.value || initializing.value) {
      return
    }

    initializing.value = true
    try {
      loadLocalSettings()
      void loadPresets()
    } finally {
      initialized.value = true
      initializing.value = false
      if (pendingPersistence) {
        persistSettings()
        pendingPersistence = false
      }
    }
  }

  // Server preferences take over once the user is authenticated; logging out
  // falls back to the guest (localStorage) state.
  if (import.meta.client) {
    watch(
      () => userStore.isAuthenticated,
      (authenticated) => {
        if (authenticated && !serverSynced.value) {
          void loadFromServer()
        } else if (!authenticated && serverSynced.value) {
          revertToGuest()
        }
      },
      { immediate: true }
    )
  }

  function selectPreset(id: string): void {
    if (!presets.value.some((preset) => preset.id === id)) {
      return
    }

    settings.value.source = 'preset'
    settings.value.presetId = id
    schedulePersistence()
  }

  function useCustomImage(): void {
    if (!settings.value.customImageId) {
      return
    }

    settings.value.source = 'custom'
    schedulePersistence()
  }

  async function uploadCustomImage(file: File): Promise<void> {
    if (!userStore.isAuthenticated) {
      throw new Error('登录后才能上传自定义背景')
    }

    const { uploadImage } = useImageUpload()
    const asset = await uploadImage(file, 'backgrounds')

    customImageUrl.value = asset.url
    settings.value.customImageId = asset.id
    settings.value.source = 'custom'
    schedulePersistence()
  }

  function setOpacity(value: number): void {
    settings.value.opacity = clampOpacity(value)
    schedulePersistence()
  }

  function setBlur(value: number): void {
    settings.value.blur = clampBlur(value)
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

  function deleteCustomImage(): void {
    customImageUrl.value = null
    settings.value.customImageId = null
    if (settings.value.source === 'custom') {
      settings.value.source = 'none'
    }
    schedulePersistence()
  }

  function reset(): void {
    settings.value = createDefaultSettings()
    customImageUrl.value = null
    schedulePersistence()
  }

  function dispose(): void {
    if (persistenceTimer) {
      clearTimeout(persistenceTimer)
      persistSettings()
      persistenceTimer = null
    }
    customImageUrl.value = null
  }

  return {
    settings,
    initialized,
    presets,
    customImageUrl,
    activePreset,
    backgroundUrl,
    hasBackground,
    initialize,
    selectPreset,
    useCustomImage,
    uploadCustomImage,
    setOpacity,
    setBlur,
    setPosition,
    setSize,
    disableBackground,
    deleteCustomImage,
    reset,
    dispose
  }
})
