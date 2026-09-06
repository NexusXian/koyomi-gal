<script setup lang="ts">
import type { DtoGalgameCharacterResponse } from '~/api/generated/models'
import { CHARACTER_ROLES } from '~/constants/character'

const props = defineProps<{
  item: DtoGalgameCharacterResponse
  revealed: boolean
  disabled: boolean
}>()
defineEmits<{ reveal: [] }>()
const concealed = computed(() => props.item.appearance_spoiler && !props.revealed)
</script>

<template>
  <article class="character-card">
    <template v-if="concealed">
      <div class="concealed-identity">???</div>
      <div class="character-info">
        <h4>隐藏角色</h4>
        <p>角色的登场本身可能涉及剧透。</p>
        <KunButton size="sm" variant="bordered" :disabled="disabled" @click="$emit('reveal')">显示角色</KunButton>
      </div>
    </template>
    <template v-else>
      <img v-if="item.image_url" :src="item.image_url" :alt="item.name" class="character-image" loading="lazy" referrerpolicy="no-referrer" />
      <div v-else class="image-placeholder"><KunIcon name="lucide:user-round" /></div>
      <div class="character-info">
        <h4>{{ item.name }}</h4>
        <p v-if="item.original_name" class="original-name">{{ item.original_name }}</p>
        <KunChip size="sm" variant="flat">{{ CHARACTER_ROLES.find((role) => role.value === item.role)?.label ?? '其他' }}</KunChip>
        <p v-if="item.public_description">{{ item.public_description }}</p>
        <p v-if="item.description">{{ item.description }}</p>
        <template v-if="item.has_spoiler">
          <p v-if="revealed && item.spoiler_description" class="spoiler-text">{{ item.spoiler_description }}</p>
          <p v-else-if="!revealed" class="spoiler-hint">包含剧透内容</p>
          <KunButton v-if="!revealed" size="sm" variant="bordered" :disabled="disabled" @click="$emit('reveal')">显示剧透内容</KunButton>
        </template>
      </div>
    </template>
  </article>
</template>

<style scoped>
.character-card { display: flex; align-items: flex-start; gap: 14px; padding: 14px; border: 1px solid var(--color-default-200); border-radius: var(--radius-kun-lg); }
.character-image, .image-placeholder, .concealed-identity { flex: 0 0 88px; width: 88px; height: 120px; border-radius: var(--radius-kun-md); object-fit: cover; }
.image-placeholder, .concealed-identity { display: grid; place-items: center; background: var(--color-content2); color: var(--color-default-500); font-size: 24px; }
.character-info { min-width: 0; overflow-wrap: anywhere; }
.character-info h4 { margin: 0 0 4px; font-size: 16px; }
.character-info p { margin: 8px 0; white-space: pre-wrap; color: var(--color-default-600); font-size: 14px; line-height: 1.7; }
.character-info .original-name { margin-top: 0; color: var(--color-default-500); }
.character-info .spoiler-hint { color: var(--color-warning); }
.spoiler-text { padding: 10px; border-left: 2px solid var(--color-warning); background: var(--color-content2); }
</style>
