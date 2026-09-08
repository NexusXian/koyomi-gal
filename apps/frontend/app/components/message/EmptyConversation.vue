<template>
  <div class="empty-conversation">
    <KunIcon name="lucide:inbox" />
    <p>选择一个会话开始聊天</p>
    <span>或从其他用户主页发起私信</span>
  </div>
</template>

<style scoped>
.empty-conversation {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--color-default-400);
}

.empty-conversation .iconify {
  font-size: 42px;
}

.empty-conversation p {
  margin: 0;
  color: var(--color-default-600);
  font-size: 15px;
  font-weight: 600;
}

.empty-conversation span {
  font-size: 13px;
}
</style>
