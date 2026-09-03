import { listGalgameGallery } from '~/api/generated/galgames/galgames'
import type { DtoGalleryListData } from '~/api/generated/models'

/**
 * Loads the CG / screenshot gallery of a galgame.
 * Gallery failures are contained here so the detail page keeps rendering.
 */
export function useGameGallery(galgameId: MaybeRefOrGetter<number>) {
  const id = computed(() => toValue(galgameId))

  const { data, pending, error, refresh } = useAsyncData<DtoGalleryListData, Error>(
    () => `galgame-gallery-${id.value}`,
    async () =>
      unwrapApiData(await listGalgameGallery(id.value), '加载游戏画面失败'),
    { watch: [id] }
  )

  const items = computed(() => data.value?.items ?? [])

  return { items, pending, error, refresh }
}
