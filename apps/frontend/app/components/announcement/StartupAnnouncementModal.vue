<script setup lang="ts">
const route = useRoute()
const { current, visible, load, acknowledge } = useAnnouncements()

onMounted(() => {
  if (!route.path.startsWith('/admin')) {
    void load()
  }
})
</script>

<template>
  <a-modal
    v-if="current && !route.path.startsWith('/admin')"
    :open="visible"
    :title="current.title"
    :closable="current.dismissible"
    :mask-closable="current.dismissible"
    :keyboard="current.dismissible"
    :width="680"
    @cancel="acknowledge"
  >
    <PostContent :content="current.content" mode="markdown" />

    <template #footer>
      <a-button type="primary" @click="acknowledge">我知道了</a-button>
    </template>
  </a-modal>
</template>
