<script setup lang="ts">
import { message as antMessage } from 'ant-design-vue'
import {
  blockMessageUser,
  unblockMessageUser
} from '~/api/generated/messages/messages'
import type { DtoConversationData } from '~/api/generated/models'

const props = defineProps<{ conversation: DtoConversationData }>()
const emit = defineEmits<{ refresh: [] }>()

const router = useRouter()
const messageStore = useMessageStore()
const blocking = ref(false)

const peer = computed(() => props.conversation.user)

function openProfile(): void {
  if (peer.value?.username) {
    void router.push(`/user/${peer.value.username}`)
  }
}

async function toggleBlock(): Promise<void> {
  if (!peer.value?.id || blocking.value) return
  blocking.value = true
  try {
    if (props.conversation.is_blocked) {
      await unblockMessageUser(peer.value.id)
      antMessage.success('已取消拉黑')
    } else {
      await blockMessageUser(peer.value.id)
      antMessage.success('已拉黑该用户')
    }
    emit('refresh')
  } catch (error) {
    antMessage.error(getApiErrorMessage(error, '拉黑设置失败'))
  } finally {
    blocking.value = false
  }
}

async function removeConversation(): Promise<void> {
  if (!props.conversation.id) return
  try {
    await messageStore.deleteConversation(props.conversation.id)
    antMessage.success('会话已删除')
    void router.replace('/messages')
  } catch (error) {
    antMessage.error(getApiErrorMessage(error, '删除会话失败'))
  }
}

function handleMenu({ key }: { key: string | number }): void {
  if (key === 'profile') openProfile()
  else if (key === 'block') void toggleBlock()
}
</script>

<template>
  <div class="message-header">
    <button type="button" class="header-peer" @click="openProfile">
      <UserAvatar
        :avatar-url="peer?.avatar_url"
        :display-name="peer?.display_name"
        :username="peer?.username"
        size="sm"
      />
      <span class="header-name">
        {{ peer?.display_name || peer?.username || '用户' }}
      </span>
    </button>

    <a-dropdown>
      <KunButton
        color="default"
        variant="light"
        size="sm"
        rounded="full"
        :is-icon-only="true"
        aria-label="会话操作"
      >
        <KunIcon name="lucide:ellipsis" />
      </KunButton>
      <template #overlay>
        <a-menu @click="handleMenu">
          <a-menu-item key="profile">
            <span class="header-menu-item">
              <KunIcon name="lucide:user-round" />查看用户主页
            </span>
          </a-menu-item>
          <a-menu-item key="block" :disabled="blocking">
            <span class="header-menu-item">
              <KunIcon name="lucide:ban" />
              {{ conversation.is_blocked ? '取消拉黑' : '拉黑用户' }}
            </span>
          </a-menu-item>
          <a-menu-divider />
          <a-popconfirm
            title="确定从你的会话列表中删除该聊天吗？"
            description="对方不受影响，收到新消息后会自动恢复。"
            ok-text="删除"
            cancel-text="取消"
            @confirm="removeConversation"
          >
            <a-menu-item key="delete" danger>
              <span class="header-menu-item">
                <KunIcon name="lucide:trash-2" />删除会话
              </span>
            </a-menu-item>
          </a-popconfirm>
        </a-menu>
      </template>
    </a-dropdown>
  </div>
</template>

<style scoped>
.message-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--app-glass-border);
}

.header-peer {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  border: 0;
  padding: 4px;
  border-radius: var(--radius-kun-md);
  background: transparent;
  cursor: pointer;
}

.header-peer:hover {
  background: color-mix(in srgb, var(--color-primary) 10%, transparent);
}

.header-name {
  overflow: hidden;
  font-weight: 700;
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--color-default-800);
}

.header-menu-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
</style>
