<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  bindGalgameCharacter,
  createCharacter,
  listAdminGalgameCharacters,
  unbindGalgameCharacter,
  updateCharacter,
  updateGalgameCharacter
} from '~/api/generated/admin/admin'
import type {
  DtoCharacterRequest,
  DtoCharacterResponse,
  DtoGalgameCharacterResponse,
  DtoUpdateGalgameCharacterRequest
} from '~/api/generated/models'
import { CHARACTER_ROLES, CHARACTER_SPOILER_LEVELS } from '~/constants/character'
import type { ImageAsset } from '~/types/image'

const props = defineProps<{ galgameId: number }>()
const { has } = usePermissions()
const items = ref<DtoGalgameCharacterResponse[]>([])
const loading = ref(false)
const error = ref('')
const busy = ref(false)
const sortOrders = ref<Record<number, number>>({})
const relationOpen = ref(false)
const editing = ref<DtoGalgameCharacterResponse>()
const selected = ref<DtoCharacterResponse>()
const sharedOpen = ref(false)
const sharedId = ref<number>()
const sharedSaving = ref(false)
let active = true
let version = 0
let controller: AbortController | undefined

const relation = reactive<Required<DtoUpdateGalgameCharacterRequest>>({
  role: 'other', spoiler_level: 'none', appearance_spoiler: false,
  description: '', spoiler_description: '', sort_order: 0
})
const shared = reactive<Required<DtoCharacterRequest>>({
  name: '', original_name: '', description: '', image_url: '', gender: '',
  birthday: '', blood_type: '', height: 0, source: '', source_id: ''
})
const columns = [
  { title: '角色', key: 'character', width: 210 },
  { title: '定位', key: 'role', width: 100 },
  { title: '剧透等级', key: 'spoiler', width: 110 },
  { title: '登场剧透', key: 'appearance', width: 100 },
  { title: '排序', key: 'sort', width: 180 },
  { title: '操作', key: 'actions', width: 260 }
]

async function load(): Promise<void> {
  const current = ++version
  const id = props.galgameId
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  error.value = ''
  try {
    const data = unwrapApiData(await listAdminGalgameCharacters(id, {
      signal: controller.signal, cache: 'no-store'
    }))
    if (!active || current !== version || id !== props.galgameId) return
    items.value = data.items ?? []
    sortOrders.value = Object.fromEntries(items.value.map((item) => [item.id ?? 0, item.sort_order ?? 0]))
  } catch (cause) {
    if (active && current === version) error.value = getApiErrorMessage(cause, '加载角色关联失败')
  } finally {
    if (active && current === version) loading.value = false
  }
}

function relationPayload(item?: DtoGalgameCharacterResponse): Required<DtoUpdateGalgameCharacterRequest> {
  return {
    role: item?.role ?? 'other', spoiler_level: item?.spoiler_level ?? 'none',
    appearance_spoiler: item?.appearance_spoiler ?? false,
    description: item?.description ?? '', spoiler_description: item?.spoiler_description ?? '',
    sort_order: item?.sort_order ?? 0
  }
}

function openRelation(item?: DtoGalgameCharacterResponse): void {
  editing.value = item
  selected.value = undefined
  Object.assign(relation, relationPayload(item))
  relationOpen.value = true
}

async function saveRelation(): Promise<void> {
  const characterId = editing.value?.character_id ?? selected.value?.id
  if (!characterId) {
    message.warning('请选择或创建角色')
    return
  }
  if (!Number.isInteger(relation.sort_order) || relation.sort_order < -2147483648 || relation.sort_order > 2147483647) {
    message.warning('排序必须是有效整数')
    return
  }
  if (busy.value || !has('character:manage')) return
  busy.value = true
  try {
    if (editing.value) {
      await updateGalgameCharacter(props.galgameId, characterId, { ...relation })
    } else {
      await bindGalgameCharacter(props.galgameId, { character_id: characterId, ...relation })
    }
    if (!active) return
    message.success('角色关联已保存')
    relationOpen.value = false
    await load()
  } catch (cause) {
    if (active) message.error(getApiErrorMessage(cause, '保存角色关联失败'))
  } finally {
    if (active) busy.value = false
  }
}

function openShared(item?: DtoGalgameCharacterResponse): void {
  sharedId.value = item?.character_id
  Object.assign(shared, {
    name: item?.name ?? '', original_name: item?.original_name ?? '',
    description: item?.public_description ?? '', image_url: item?.image_url ?? '',
    gender: item?.gender ?? '', birthday: item?.birthday ?? '', blood_type: item?.blood_type ?? '',
    height: item?.height ?? 0, source: item?.source ?? '', source_id: item?.source_id ?? ''
  })
  sharedOpen.value = true
}

function imageUploaded(asset: ImageAsset): void {
  if (active && sharedOpen.value) shared.image_url = asset.url
}

async function saveShared(): Promise<void> {
  if (!shared.name.trim()) {
    message.warning('请输入角色名称')
    return
  }
  if (!Number.isInteger(shared.height) || shared.height < 0 || shared.height > 2147483647) {
    message.warning('身高必须为非负整数，未知请填写 0')
    return
  }
  const imageUrl = shared.image_url.trim()
  if (imageUrl) {
    try {
      const url = new URL(imageUrl)
      if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password || /\s/.test(imageUrl)) throw new Error()
    } catch {
      message.warning('图片地址须为不含账号、密码或空白的 HTTP/HTTPS URL')
      return
    }
  }
  if (sharedSaving.value || !has('character:manage')) return
  sharedSaving.value = true
  try {
    const payload = { ...shared, name: shared.name.trim(), image_url: imageUrl }
    const data = unwrapApiData(sharedId.value
      ? await updateCharacter(sharedId.value, payload)
      : await createCharacter(payload))
    if (!active) return
    if (!sharedId.value) {
      selected.value = data
      message.success('角色已创建或复用，请保存与本游戏的关联')
    } else {
      message.success('共享角色资料已更新')
    }
    sharedOpen.value = false
    await load()
  } catch (cause) {
    if (active) message.error(getApiErrorMessage(cause, '保存角色资料失败'))
  } finally {
    if (active) sharedSaving.value = false
  }
}

async function saveSort(item: DtoGalgameCharacterResponse): Promise<void> {
  const sortOrder = sortOrders.value[item.id ?? 0]
  if (!item.character_id || busy.value || !has('character:manage')) return
  if (sortOrder === undefined || !Number.isInteger(sortOrder) || sortOrder < -2147483648 || sortOrder > 2147483647) {
    message.warning('排序必须是有效整数')
    return
  }
  busy.value = true
  try {
    await updateGalgameCharacter(props.galgameId, item.character_id, { ...relationPayload(item), sort_order: sortOrder })
    if (!active) return
    message.success('排序已保存')
    await load()
  } catch (cause) {
    if (active) message.error(getApiErrorMessage(cause, '保存排序失败'))
  } finally {
    if (active) busy.value = false
  }
}

async function unlink(item: DtoGalgameCharacterResponse): Promise<void> {
  if (!item.character_id || busy.value || !has('character:manage')) return
  busy.value = true
  try {
    await unbindGalgameCharacter(props.galgameId, item.character_id)
    if (!active) return
    message.success('已解除关联，共享角色未被删除')
    await load()
  } catch (cause) {
    if (active) message.error(getApiErrorMessage(cause, '解除关联失败'))
  } finally {
    if (active) busy.value = false
  }
}

onMounted(() => void load())
onBeforeUnmount(() => {
  active = false
  ++version
  controller?.abort()
  relationOpen.value = false
  sharedOpen.value = false
})
</script>

<template>
  <KunCard padding="lg" class-name="character-manager">
    <div class="manager-header">
      <KunHeader name="角色管理" description="管理本游戏的角色关联；此处包含所有剧透及未发布条目的角色。" scale="h3" />
      <KunButton size="sm" :disabled="busy || loading" @click="openRelation()">
        <KunIcon name="lucide:plus" />关联角色
      </KunButton>
    </div>
    <a-alert v-if="error" type="error" show-icon :message="error">
      <template #action><a-button size="small" @click="load">重试</a-button></template>
    </a-alert>
    <a-table :columns="columns" :data-source="items" row-key="id" :loading="loading || busy" :pagination="false" :scroll="{ x: 960 }">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'character'">
          <strong>{{ record.name }}</strong>
          <p class="original-name">{{ record.original_name }}</p>
        </template>
        <template v-else-if="column.key === 'role'">{{ CHARACTER_ROLES.find((item) => item.value === record.role)?.label }}</template>
        <template v-else-if="column.key === 'spoiler'">{{ CHARACTER_SPOILER_LEVELS.find((item) => item.value === record.spoiler_level)?.label }}</template>
        <template v-else-if="column.key === 'appearance'">
          <KunChip size="sm" :color="record.appearance_spoiler ? 'warning' : 'default'" variant="flat">{{ record.appearance_spoiler ? '隐藏登场' : '公开' }}</KunChip>
        </template>
        <template v-else-if="column.key === 'sort'">
          <div class="row-actions">
            <a-input-number v-model:value="sortOrders[record.id]" :min="-2147483648" :max="2147483647" :precision="0" :disabled="busy || loading" aria-label="角色排序" />
            <a-button size="small" :disabled="busy || loading || sortOrders[record.id] === record.sort_order" @click="saveSort(record)">保存</a-button>
          </div>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="row-actions">
            <a-button size="small" :disabled="busy || loading" @click="openRelation(record)">编辑关联</a-button>
            <a-button size="small" :disabled="busy || loading" @click="openShared(record)">编辑资料</a-button>
            <a-popconfirm title="仅解除本游戏与该角色的关联？共享角色及其他游戏的关联不会被删除。" ok-text="解除关联" cancel-text="取消" :disabled="busy || loading" @confirm="unlink(record)">
              <a-button size="small" danger :disabled="busy || loading">解除关联</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal
      v-model:open="relationOpen"
      :title="editing ? `编辑关联：${editing.name}` : '关联角色'"
      :confirm-loading="busy"
      :closable="!busy"
      :mask-closable="!busy"
      :keyboard="!busy"
      :cancel-button-props="{ disabled: busy }"
      ok-text="保存关联"
      cancel-text="取消"
      destroy-on-close
      @ok="saveRelation"
    >
      <a-form layout="vertical" :disabled="busy">
        <a-form-item v-if="!editing" label="已有角色" required>
          <CharacterSelector v-if="relationOpen" v-model="selected" :disabled="busy" />
          <a-button class="create-button" @click="openShared()">创建角色</a-button>
        </a-form-item>
        <div class="form-grid">
          <a-form-item label="角色定位"><a-select v-model:value="relation.role" :options="CHARACTER_ROLES" /></a-form-item>
          <a-form-item label="剧透等级"><a-select v-model:value="relation.spoiler_level" :options="CHARACTER_SPOILER_LEVELS" /></a-form-item>
          <a-form-item label="登场本身涉及剧透"><a-switch v-model:checked="relation.appearance_spoiler" /></a-form-item>
          <a-form-item label="排序（越小越靠前）"><a-input-number v-model:value="relation.sort_order" :min="-2147483648" :max="2147483647" :precision="0" /></a-form-item>
        </div>
        <a-form-item label="本游戏中的角色描述（无剧透）"><a-textarea v-model:value="relation.description" :rows="3" /></a-form-item>
        <a-form-item label="剧透描述（确认显示后才会公开）"><a-textarea v-model:value="relation.spoiler_description" :rows="4" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="sharedOpen"
      :title="sharedId ? '编辑共享角色资料' : '创建角色'"
      :confirm-loading="sharedSaving"
      :closable="!sharedSaving"
      :mask-closable="!sharedSaving"
      :keyboard="!sharedSaving"
      :cancel-button-props="{ disabled: sharedSaving }"
      ok-text="保存资料"
      cancel-text="取消"
      destroy-on-close
      @ok="saveShared"
    >
      <a-alert type="warning" show-icon :message="sharedId ? '共享资料的修改会影响所有关联游戏。共享简介只能填写无剧透内容。' : '共享简介只能填写无剧透内容；相同来源及来源 ID 会复用已有角色，不覆盖资料。'" />
      <a-form layout="vertical" :disabled="sharedSaving" class="shared-form">
        <div class="form-grid">
          <a-form-item label="名称" required><a-input v-model:value="shared.name" :maxlength="255" /></a-form-item>
          <a-form-item label="原名"><a-input v-model:value="shared.original_name" :maxlength="255" /></a-form-item>
          <a-form-item label="性别"><a-input v-model:value="shared.gender" :maxlength="50" /></a-form-item>
          <a-form-item label="生日"><a-input v-model:value="shared.birthday" placeholder="例如：03-14" :maxlength="50" /></a-form-item>
          <a-form-item label="血型"><a-input v-model:value="shared.blood_type" :maxlength="50" /></a-form-item>
          <a-form-item label="身高（cm，0 为未知）"><a-input-number v-model:value="shared.height" :min="0" :max="2147483647" :precision="0" /></a-form-item>
          <a-form-item label="来源"><a-input v-model:value="shared.source" placeholder="例如：vndb" :maxlength="50" /></a-form-item>
          <a-form-item label="来源 ID"><a-input v-model:value="shared.source_id" placeholder="例如：c123" :maxlength="255" /></a-form-item>
        </div>
        <a-form-item label="共享简介（禁止剧透）"><a-textarea v-model:value="shared.description" :rows="4" /></a-form-item>
        <a-form-item label="角色图片">
          <ImageUploader
            v-if="sharedOpen && has('image:manage')"
            category="galgames"
            :preview-url="shared.image_url || null"
            :disabled="sharedSaving"
            width="84px"
            height="112px"
            @success="imageUploaded"
            @remove="shared.image_url = ''"
          />
          <a-input v-model:value="shared.image_url" :maxlength="2048" placeholder="HTTP/HTTPS 外部图片 URL，或上传后自动填充" />
        </a-form-item>
      </a-form>
    </a-modal>
  </KunCard>
</template>

<style scoped>
.character-manager { margin-top: 18px; }
.manager-header, .row-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.manager-header { justify-content: space-between; margin-bottom: 16px; }
.original-name { margin: 4px 0 0; color: var(--color-default-500); font-size: 13px; overflow-wrap: anywhere; }
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 16px; }
.create-button, .shared-form { margin-top: 12px; }
@media (max-width: 575px) { .form-grid { grid-template-columns: minmax(0, 1fr); } }
</style>
