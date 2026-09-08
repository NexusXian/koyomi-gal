<script setup lang="ts">
import { message } from 'ant-design-vue'
import { changePassword, passwordChangeCode } from '~/api/generated/users/users'

useSeoMeta({ title: '账号与安全 - Koyomi' })

const router = useRouter()
const { logout } = useAuth()
const userStore = useUserStore()
const { user, initialized, isAuthenticated } = storeToRefs(userStore)
const code = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const sendingCode = ref(false)
const saving = ref(false)
const codeSentTo = ref('')
const { cooldown, start: startCooldown } = useSendCooldown(60)

const maskedEmail = computed(() => {
  const email = user.value?.email ?? ''
  const [local, domain] = email.split('@')
  if (!local || !domain) {
    return ''
  }
  const visible = local.length < 3 ? 1 : 2
  return `${local.slice(0, visible)}***@${domain}`
})

const sendCodeLabel = computed(() => {
  if (sendingCode.value) {
    return '发送中...'
  }
  return cooldown.value > 0 ? `${cooldown.value} 秒后重试` : '获取验证码'
})

async function sendCode(): Promise<void> {
  if (sendingCode.value || cooldown.value > 0) {
    return
  }

  sendingCode.value = true
  try {
    const data = unwrapApiData(await passwordChangeCode(), '验证码发送失败')
    codeSentTo.value = data.email || maskedEmail.value
    startCooldown()
    message.success('验证码已发送，请查收邮箱')
  } catch (error: unknown) {
    message.error(getApiErrorMessage(error, '验证码发送失败，请稍后重试'))
  } finally {
    sendingCode.value = false
  }
}

async function submitChange(): Promise<void> {
  if (saving.value) {
    return
  }

  if (!/^\d{6}$/.test(code.value)) {
    message.error('请输入 6 位数字验证码')
    return
  }
  if (newPassword.value.length < 8 || newPassword.value.length > 72) {
    message.error('密码长度必须为 8 到 72 个字符')
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    message.error('两次输入的密码不一致')
    return
  }

  saving.value = true
  try {
    await changePassword({
      code: code.value,
      new_password: newPassword.value,
      confirm_password: confirmPassword.value
    })
    message.success('密码修改成功，请重新登录')
    try {
      await logout()
    } catch {
      // 本地登录状态已清除，登出接口失败不阻断跳转
    }
    await router.replace('/login')
  } catch (error: unknown) {
    message.error(getApiErrorMessage(error, '密码修改失败，请稍后重试'))
  } finally {
    saving.value = false
  }
}

watch(
  [initialized, isAuthenticated],
  ([ready, authenticated]) => {
    if (!ready) {
      return
    }
    if (!authenticated) {
      void router.replace('/login')
    }
  },
  { immediate: true }
)
</script>

<template>
  <AppPageContainer title="账号与安全" description="通过邮箱验证码修改账号密码。">
    <nav class="settings-nav" aria-label="设置导航">
      <NuxtLink to="/settings/profile">个人资料</NuxtLink>
      <NuxtLink to="/settings/privacy">隐私设置</NuxtLink>
      <NuxtLink class="active" to="/settings/security">账号与安全</NuxtLink>
      <NuxtLink v-if="user?.username" :to="`/user/${user.username}`">查看个人空间</NuxtLink>
    </nav>

    <KunCard padding="lg">
      <KunHeader
        name="修改密码"
        description="验证码将发送到当前账号绑定的邮箱"
        scale="h3"
      />
      <a-form layout="vertical" @submit.prevent="submitChange">
        <div class="email-row">
          <div>
            <strong>绑定邮箱</strong>
            <p>{{ codeSentTo || maskedEmail || '当前账号未提供邮箱' }}</p>
          </div>
          <a-button
            :loading="sendingCode"
            :disabled="cooldown > 0"
            @click="sendCode"
          >
            {{ sendCodeLabel }}
          </a-button>
        </div>

        <a-form-item label="邮箱验证码">
          <a-input
            v-model:value="code"
            placeholder="请输入 6 位验证码"
            :maxlength="6"
            autocomplete="one-time-code"
          />
        </a-form-item>
        <a-form-item label="新密码">
          <a-input-password
            v-model:value="newPassword"
            placeholder="8-72 个字符"
            :maxlength="72"
            autocomplete="new-password"
          />
        </a-form-item>
        <a-form-item label="确认新密码">
          <a-input-password
            v-model:value="confirmPassword"
            placeholder="请再次输入新密码"
            :maxlength="72"
            autocomplete="new-password"
          />
        </a-form-item>
        <div class="save-row">
          <a-button
            type="primary"
            html-type="submit"
            :loading="saving"
            :disabled="sendingCode"
          >
            确认修改
          </a-button>
        </div>
      </a-form>
    </KunCard>
  </AppPageContainer>
</template>

<style scoped>
.settings-nav { display: flex; flex-wrap: wrap; gap: 6px; margin-bottom: 16px; }
.settings-nav a { padding: 8px 14px; border-radius: var(--radius-kun-md); color: var(--color-default-500); }
.settings-nav a:hover, .settings-nav a.active { background: color-mix(in srgb, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.email-row { display: flex; align-items: center; justify-content: space-between; gap: 24px; padding: 12px 0 20px; border-bottom: 1px solid var(--color-default-200); margin-bottom: 20px; }
.email-row strong { font-size: 15px; }
.email-row p { margin: 4px 0 0; color: var(--color-default-500); font-size: 13px; }
.save-row { display: flex; justify-content: flex-end; }
@media (max-width: 480px) {
  .email-row { align-items: flex-start; flex-direction: column; }
}
</style>
