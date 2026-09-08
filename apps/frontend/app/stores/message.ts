import { defineStore } from 'pinia'
import {
  createMessageConversation,
  deleteMessage,
  deleteMessageConversation,
  getMessageUnreadCount,
  listConversationMessages,
  listMessageConversations,
  markConversationRead,
  sendConversationMessage
} from '~/api/generated/messages/messages'
import { createWebSocketTicket } from '~/api/generated/websocket/websocket'
import type {
  DtoConversationData,
  DtoMessageData,
  DtoMessagePreview
} from '~/api/generated/models'

export type ChatMessageStatus = 'sending' | 'failed'

export interface ChatMessage extends DtoMessageData {
  clientTempId?: string
  status?: ChatMessageStatus
}

export interface RealtimeEvent {
  type?: string
  data?: Record<string, unknown>
}

interface MessageState {
  conversations: DtoConversationData[]
  conversationsLoading: boolean
  conversationsError: string
  conversationCursor: string
  hasMoreConversations: boolean
  currentConversationId: number | null
  messages: Record<number, ChatMessage[]>
  messagesLoading: Record<number, boolean>
  olderLoading: Record<number, boolean>
  hasMoreMessages: Record<number, boolean>
  unreadCount: number
  wsConnected: boolean
}

let ws: WebSocket | undefined
let reconnectAttempts = 0
let reconnectTimer: ReturnType<typeof setTimeout> | undefined
let wsClosedByUs = false

export const useMessageStore = defineStore('message', {
  state: (): MessageState => ({
    conversations: [],
    conversationsLoading: false,
    conversationsError: '',
    conversationCursor: '',
    hasMoreConversations: false,
    currentConversationId: null,
    messages: {},
    messagesLoading: {},
    olderLoading: {},
    hasMoreMessages: {},
    unreadCount: 0,
    wsConnected: false
  }),

  getters: {
    currentConversation(state): DtoConversationData | undefined {
      if (state.currentConversationId === null) return undefined
      return state.conversations.find(
        (conversation) => conversation.id === state.currentConversationId
      )
    },

    currentMessages(state): ChatMessage[] {
      if (state.currentConversationId === null) return []
      return state.messages[state.currentConversationId] ?? []
    }
  },

  actions: {
    setUnreadCount(count: number): void {
      this.unreadCount = Math.max(0, count)
    },

    async fetchUnreadCount(): Promise<void> {
      const userStore = useUserStore()
      if (!userStore.isAuthenticated) {
        this.unreadCount = 0
        return
      }
      try {
        const data = unwrapApiData(
          await getMessageUnreadCount(),
          '查询未读私信失败'
        )
        this.setUnreadCount(data.count ?? 0)
      } catch {
        // 下次聚焦或轮询时重试
      }
    },

    async fetchConversations(reset = true): Promise<void> {
      if (this.conversationsLoading) return
      this.conversationsLoading = true
      if (reset) {
        this.conversationsError = ''
      }
      try {
        const data = unwrapApiData(
          await listMessageConversations(
            reset ? undefined : { cursor: this.conversationCursor || undefined }
          ),
          '查询私信会话失败'
        )
        const list = data.list ?? []
        this.conversations = reset ? list : [...this.conversations, ...list]
        this.conversationCursor = data.next_cursor ?? ''
        this.hasMoreConversations = Boolean(data.has_more)
      } catch (error) {
        if (reset) {
          this.conversationsError = getApiErrorMessage(error, '查询私信会话失败')
        }
      } finally {
        this.conversationsLoading = false
      }
    },

    async loadMoreConversations(): Promise<void> {
      if (!this.hasMoreConversations || this.conversationsLoading) return
      await this.fetchConversations(false)
    },

    async startConversationWith(userId: number): Promise<number> {
      const data = unwrapApiData(
        await createMessageConversation({ user_id: userId }),
        '发起私信失败'
      )
      const conversation = data
      const existing = this.conversations.find(
        (item) => item.id === conversation.id
      )
      if (existing) {
        Object.assign(existing, conversation)
      } else {
        this.conversations.unshift(conversation)
      }
      return conversation.id ?? 0
    },

    async fetchMessages(conversationId: number, beforeId?: number): Promise<void> {
      if (beforeId) {
        if (this.olderLoading[conversationId]) return
      } else if (this.messagesLoading[conversationId]) {
        return
      }
      if (beforeId) {
        this.olderLoading[conversationId] = true
      } else {
        this.messagesLoading[conversationId] = true
      }
      try {
        const data = unwrapApiData(
          await listConversationMessages(
            conversationId,
            beforeId ? { before_id: beforeId } : undefined
          ),
          '查询聊天记录失败'
        )
        const list = (data.list ?? []) as ChatMessage[]
        const current = this.messages[conversationId] ?? []
        const merged = beforeId ? [...list, ...current] : list
        this.messages[conversationId] = dedupeMessages(merged)
        this.hasMoreMessages[conversationId] = Boolean(data.has_more)
      } catch {
        // 页面会展示重试入口
      } finally {
        this.olderLoading[conversationId] = false
        this.messagesLoading[conversationId] = false
      }
    },

    async openConversation(conversationId: number): Promise<void> {
      this.currentConversationId = conversationId
      if (!this.messages[conversationId]) {
        await this.fetchMessages(conversationId)
      }
      const messages = this.messages[conversationId] ?? []
      const latest = [...messages]
        .filter((item) => item.id && !item.status)
        .sort((a, b) => (a.id ?? 0) - (b.id ?? 0))
        .pop()
      if (latest?.id) {
        await this.markRead(conversationId, latest.id)
      }
    },

    closeConversation(): void {
      this.currentConversationId = null
    },

    async markRead(conversationId: number, messageId: number): Promise<void> {
      try {
        await markConversationRead(conversationId, { message_id: messageId })
      } catch {
        return
      }
      const conversation = this.conversations.find(
        (item) => item.id === conversationId
      )
      if (conversation?.unread_count) {
        this.unreadCount = Math.max(
          0,
          this.unreadCount - (conversation.unread_count ?? 0)
        )
        conversation.unread_count = 0
      }
    },

    async sendMessage(conversationId: number, content: string): Promise<void> {
      const userStore = useUserStore()
      const trimmed = content.trim()
      if (!trimmed) return
      const clientTempId = `local-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`
      const optimistic: ChatMessage = {
        clientTempId,
        status: 'sending',
        conversation_id: conversationId,
        sender_id: userStore.user?.id ?? undefined,
        sender: undefined,
        type: 'text',
        content: trimmed,
        is_deleted: false,
        created_at: new Date().toISOString()
      }
      const list = this.messages[conversationId] ?? []
      this.messages[conversationId] = [...list, optimistic]

      try {
        const data = unwrapApiData(
          await sendConversationMessage(conversationId, {
            type: 'text',
            content: trimmed
          }),
          '发送私信失败'
        )
        this.commitServerMessage(conversationId, data, clientTempId)
      } catch (error) {
        const entries = this.messages[conversationId] ?? []
        const target = entries.find((item) => item.clientTempId === clientTempId)
        if (target) {
          target.status = 'failed'
        }
        throw error
      }
    },

    async retryMessage(conversationId: number, clientTempId: string): Promise<void> {
      const entries = this.messages[conversationId] ?? []
      const target = entries.find((item) => item.clientTempId === clientTempId)
      if (!target?.content || target.status !== 'failed') return
      target.status = 'sending'
      try {
        const data = unwrapApiData(
          await sendConversationMessage(conversationId, {
            type: 'text',
            content: target.content
          }),
          '发送私信失败'
        )
        this.commitServerMessage(conversationId, data, clientTempId)
      } catch (error) {
        target.status = 'failed'
        throw error
      }
    },

    commitServerMessage(
      conversationId: number,
      message: DtoMessageData,
      clientTempId?: string
    ): void {
      const entries = this.messages[conversationId] ?? []
      if (clientTempId) {
        const index = entries.findIndex(
          (item) => item.clientTempId === clientTempId
        )
        if (index >= 0) {
          entries.splice(index, 1, message)
          this.messages[conversationId] = dedupeMessages([...entries])
          return
        }
      }
      if (entries.some((item) => item.id && item.id === message.id)) return
      this.syncPendingByContent(conversationId, message)
      const remaining = this.messages[conversationId] ?? []
      this.messages[conversationId] = dedupeMessages([...remaining, message])
    },

    // WebSocket 先于 REST 响应到达时，移除同内容的本地乐观消息
    syncPendingByContent(conversationId: number, message: DtoMessageData): void {
      const entries = this.messages[conversationId] ?? []
      const index = entries.findIndex(
        (item) =>
          !item.id &&
          item.status === 'sending' &&
          item.sender_id === message.sender_id &&
          item.content === message.content
      )
      if (index >= 0) {
        entries.splice(index, 1)
        this.messages[conversationId] = [...entries]
      }
    },

    async deleteMessage(conversationId: number, messageId: number): Promise<void> {
      await deleteMessage(messageId)
      const entries = this.messages[conversationId] ?? []
      const target = entries.find((item) => item.id === messageId)
      if (target) {
        target.is_deleted = true
        target.content = ''
      }
      const conversation = this.conversations.find(
        (item) => item.id === conversationId
      )
      if (conversation?.last_message?.id === messageId) {
        conversation.last_message = {
          ...(conversation.last_message as DtoMessagePreview),
          is_deleted: true,
          content: ''
        }
      }
    },

    async deleteConversation(conversationId: number): Promise<void> {
      await deleteMessageConversation(conversationId)
      this.conversations = this.conversations.filter(
        (item) => item.id !== conversationId
      )
      delete this.messages[conversationId]
      if (this.currentConversationId === conversationId) {
        this.currentConversationId = null
      }
    },

    async resync(): Promise<void> {
      await Promise.all([this.fetchUnreadCount(), this.fetchConversations(true)])
      const conversationId = this.currentConversationId
      if (conversationId !== null) {
        await this.fetchMessages(conversationId)
      }
    },

    handleRealtimeEvent(event: RealtimeEvent): void {
      switch (event.type) {
        case 'message.created':
          this.handleMessageCreated(event.data)
          break
        case 'conversation.read':
          this.handleConversationRead(event.data)
          break
        case 'message.deleted':
          this.handleMessageDeleted(event.data)
          break
      }
    },

    handleMessageCreated(data?: Record<string, unknown>): void {
      const userStore = useUserStore()
      const me = userStore.user?.id
      const conversationId = Number(data?.conversation_id)
      const message = data?.message as DtoMessageData | undefined
      if (!conversationId || !message?.id) return
      const fromMe = message.sender_id === me
      const isCurrent = this.currentConversationId === conversationId

      const conversation = this.conversations.find(
        (item) => item.id === conversationId
      )
      if (conversation) {
        conversation.last_message = {
          id: message.id,
          sender_id: message.sender_id,
          type: message.type,
          content: message.content,
          is_deleted: message.is_deleted,
          created_at: message.created_at
        }
        conversation.last_message_at = message.created_at
        conversation.updated_at = message.created_at
        if (!fromMe && !isCurrent) {
          conversation.unread_count = (conversation.unread_count ?? 0) + 1
          this.unreadCount += 1
        }
        this.conversations = [
          conversation,
          ...this.conversations.filter((item) => item.id !== conversationId)
        ]
      } else if (!fromMe) {
        void this.fetchConversations(true)
      }

      if (isCurrent) {
        this.commitServerMessage(conversationId, message)
        if (!fromMe && message.id) {
          void this.markRead(conversationId, message.id)
        }
      }
    },

    handleConversationRead(data?: Record<string, unknown>): void {
      const userStore = useUserStore()
      const me = userStore.user?.id
      if (Number(data?.user_id) !== me) return
      const conversationId = Number(data?.conversation_id)
      const conversation = this.conversations.find(
        (item) => item.id === conversationId
      )
      if (conversation?.unread_count) {
        this.unreadCount = Math.max(
          0,
          this.unreadCount - (conversation.unread_count ?? 0)
        )
        conversation.unread_count = 0
      }
    },

    handleMessageDeleted(data?: Record<string, unknown>): void {
      const conversationId = Number(data?.conversation_id)
      const messageId = Number(data?.message_id)
      if (!conversationId || !messageId) return
      const entries = this.messages[conversationId]
      const target = entries?.find((item) => item.id === messageId)
      if (target) {
        target.is_deleted = true
        target.content = ''
      }
      const conversation = this.conversations.find(
        (item) => item.id === conversationId
      )
      if (conversation?.last_message?.id === messageId) {
        conversation.last_message.is_deleted = true
        conversation.last_message.content = ''
      }
    },

    connectRealtime(): void {
      if (import.meta.server || ws || reconnectTimer) return
      const userStore = useUserStore()
      if (!userStore.isAuthenticated) return

      wsClosedByUs = false
      void this.openSocket()
    },

    async openSocket(): Promise<void> {
      const userStore = useUserStore()
      if (!userStore.isAuthenticated || ws || wsClosedByUs) return
      try {
        const data = unwrapApiData(
          await createWebSocketTicket(),
          '创建 WebSocket 票据失败'
        )
        if (!data.ticket || wsClosedByUs) return
        const config = useRuntimeConfig()
        const base = String(config.public.apiBase || '').replace(/^http/, 'ws')
        const socket = new WebSocket(
          `${base}/api/v1/ws?ticket=${encodeURIComponent(data.ticket)}`
        )
        ws = socket
        socket.addEventListener('open', () => {
          reconnectAttempts = 0
          this.wsConnected = true
          void this.resync()
        })
        socket.addEventListener('message', (event) => {
          try {
            this.handleRealtimeEvent(JSON.parse(String(event.data)))
          } catch {
            // 忽略无法解析的事件
          }
        })
        socket.addEventListener('close', () => {
          ws = undefined
          this.wsConnected = false
          if (!wsClosedByUs && userStore.isAuthenticated) {
            this.scheduleReconnect()
          }
        })
        socket.addEventListener('error', () => {
          socket.close()
        })
      } catch {
        this.scheduleReconnect()
      }
    },

    scheduleReconnect(): void {
      if (wsClosedByUs || reconnectTimer) return
      const delay = Math.min(1000 * 2 ** reconnectAttempts, 30_000)
      reconnectAttempts += 1
      reconnectTimer = setTimeout(() => {
        reconnectTimer = undefined
        void this.openSocket()
      }, delay)
    },

    disconnectRealtime(): void {
      wsClosedByUs = true
      if (reconnectTimer) {
        clearTimeout(reconnectTimer)
        reconnectTimer = undefined
      }
      if (ws) {
        ws.close()
        ws = undefined
      }
      reconnectAttempts = 0
      this.wsConnected = false
    },

    resetState(): void {
      this.disconnectRealtime()
      this.$reset()
    }
  }
})

function dedupeMessages(list: ChatMessage[]): ChatMessage[] {
  const seen = new Set<number>()
  const result: ChatMessage[] = []
  for (const item of list) {
    if (item.id) {
      if (seen.has(item.id)) continue
      seen.add(item.id)
    }
    result.push(item)
  }
  return result.sort((a, b) => {
    const aId = a.id ?? Number.MAX_SAFE_INTEGER
    const bId = b.id ?? Number.MAX_SAFE_INTEGER
    if (aId !== bId) return aId - bId
    return (a.created_at ?? '') < (b.created_at ?? '') ? -1 : 1
  })
}
