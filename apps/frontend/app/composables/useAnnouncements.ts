import { createAnnouncementService } from '~/services/announcement'
import type { Announcement } from '~/types/announcement'

export function useAnnouncements() {
  const current = ref<Announcement | null>(null)
  const visible = computed(() => current.value !== null)

  function isDismissed(id: number): boolean {
    try {
      return localStorage.getItem(`announcement:${id}:dismissed`) === 'true'
    } catch {
      return false
    }
  }

  async function load(): Promise<void> {
    try {
      const announcements = await createAnnouncementService(
        useNuxtApp().$api
      ).listActive('web')
      current.value = announcements
        .filter(
          (item) =>
            item.displayMode === 'startup_modal' && !isDismissed(item.id)
        )
        .sort((a, b) => b.priority - a.priority)[0] ?? null
    } catch {
      current.value = null
    }
  }

  function acknowledge(): void {
    if (!current.value) return

    try {
      localStorage.setItem(
        `announcement:${current.value.id}:dismissed`,
        'true'
      )
    } catch {
      // Storage may be unavailable; the acknowledgement should still close it.
    }
    current.value = null
  }

  return { current, visible, load, acknowledge }
}
