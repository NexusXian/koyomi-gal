import { createPreferenceService } from '~/services/preference'
import type { SensitiveCoverMode, UserPreferences } from '~/types/background'

/**
 * Shared sensitive-cover display preference.
 *
 * Guests always default to blur; logged-in users load their
 * sensitive_cover_mode from /me/preferences once per session and may switch
 * between blur and show.
 */
export function useSensitiveCover() {
  const mode = useState<SensitiveCoverMode>('sensitive-cover-mode', () => 'blur')
  const preferences = useState<UserPreferences | null>(
    'sensitive-cover-preferences',
    () => null
  )
  const loaded = useState<boolean>('sensitive-cover-loaded', () => false)
  const userStore = useUserStore()

  function service() {
    return createPreferenceService(useNuxtApp().$api)
  }

  function applyMode(value: string | undefined | null): void {
    mode.value = value === 'show' ? 'show' : 'blur'
  }

  async function load(): Promise<void> {
    try {
      const data = await service().getPreferences()
      preferences.value = data
      applyMode(data.sensitive_cover_mode)
    } catch {
      // Keep the blur default when preferences cannot be loaded.
    } finally {
      loaded.value = true
    }
  }

  if (import.meta.client) {
    watch(
      () => userStore.isAuthenticated,
      (authenticated) => {
        if (authenticated) {
          if (!loaded.value) {
            void load()
          }
          return
        }
        mode.value = 'blur'
        preferences.value = null
        loaded.value = false
      },
      { immediate: true }
    )
  }

  async function setMode(next: SensitiveCoverMode): Promise<boolean> {
    const previous = mode.value
    mode.value = next
    try {
      const base = preferences.value ?? (await service().getPreferences())
      const updated = await service().updatePreferences({
        background_source: base.background_source,
        background_asset_id: base.background_asset_id,
        background_preset: base.background_preset,
        background_opacity: base.background_opacity,
        background_blur: base.background_blur,
        background_position: base.background_position,
        background_size: base.background_size,
        sensitive_cover_mode: next
      })
      preferences.value = updated
      applyMode(updated.sensitive_cover_mode)
      return true
    } catch {
      mode.value = previous
      return false
    }
  }

  const showSensitiveCovers = computed(() => mode.value === 'show')

  return { mode, showSensitiveCovers, setMode }
}
