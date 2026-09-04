<script setup lang="ts">
import dayjs from 'dayjs'
import { listGalgameContributors } from '~/api/generated/galgames/galgames'
import type {
  DtoContributorData,
  DtoContributorListData
} from '~/api/generated/models'

const props = withDefaults(
  defineProps<{
    galgameId: number
    contributors?: DtoContributorData[]
    contributorCount?: number
  }>(),
  { contributors: () => [], contributorCount: 0 }
)

const open = ref(false)
const loading = ref(false)
const loadFailed = ref(false)
const items = ref<DtoContributorData[]>([])
const page = ref(1)
const pageSize = 20
const total = ref(0)

const totalPage = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function formatDate(value?: string): string {
  return value ? dayjs(value).format('YYYY-MM-DD') : '未知'
}

async function loadContributors(): Promise<void> {
  loading.value = true
  loadFailed.value = false
  try {
    const data = unwrapApiData<DtoContributorListData>(
      await listGalgameContributors(props.galgameId, {
        page: page.value,
        page_size: pageSize
      })
    )
    items.value = data.items ?? []
    total.value = data.total ?? 0
  } catch {
    items.value = []
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function showAll(): void {
  open.value = true
  page.value = 1
  void loadContributors()
}

function changePage(next: number): void {
  page.value = next
  void loadContributors()
}
</script>

<template>
  <KunCard padding="lg" class-name="contributors-card">
    <div class="contributors-heading">
      <KunHeader
        :name="`贡献者 · ${contributorCount}`"
        scale="h3"
        class="section-heading"
      />
      <button
        v-if="contributorCount > contributors.length"
        type="button"
        class="view-all"
        @click="showAll"
      >
        查看全部 {{ contributorCount }} 位贡献者
        <KunIcon name="lucide:arrow-right" />
      </button>
    </div>

    <KunNull v-if="contributors.length === 0" message="暂无贡献记录" />

    <div v-else class="contributor-list">
      <a-tooltip v-for="contributor in contributors" :key="contributor.user_id">
        <template #title>
          <div class="contributor-tooltip">
            <strong>{{ contributor.username || `用户 #${contributor.user_id}` }}</strong>
            <span>{{ contributor.contribution_count ?? 0 }} 次贡献</span>
            <span>最近贡献：{{ formatDate(contributor.last_contributed_at) }}</span>
          </div>
        </template>
        <UserLink
          class="contributor-chip"
          :username="contributor.username"
          :display-name="contributor.username"
          :user-id="contributor.user_id"
          :popover="false"
        >
          <UserAvatar
            :avatar-url="contributor.avatar_url"
            :display-name="contributor.username"
            :username="contributor.username"
            size="md"
          />
          <span>{{ contributor.username || `用户 #${contributor.user_id}` }}</span>
        </UserLink>
      </a-tooltip>
    </div>

    <a-modal v-model:open="open" title="全部贡献者" :footer="null" width="640px">
      <a-spin :spinning="loading">
        <KunNull
          v-if="loadFailed"
          message="贡献者列表加载失败"
        />
        <KunNull
          v-else-if="items.length === 0"
          message="暂无贡献记录"
        />
        <ul v-else class="contributor-modal-list">
          <li v-for="contributor in items" :key="contributor.user_id">
            <UserLink
              class="contributor-identity"
              :username="contributor.username"
              :display-name="contributor.username"
              :user-id="contributor.user_id"
            >
              <UserAvatar
                :avatar-url="contributor.avatar_url"
                :display-name="contributor.username"
                :username="contributor.username"
                size="md"
              />
              <span>
                <strong>{{ contributor.username || `用户 #${contributor.user_id}` }}</strong>
                <small>ID: {{ contributor.user_id }}</small>
              </span>
            </UserLink>
            <span class="contributor-stats">
              <strong>{{ contributor.contribution_count ?? 0 }} 次贡献</strong>
              <small>首次：{{ formatDate(contributor.first_contributed_at) }}</small>
              <small>最近：{{ formatDate(contributor.last_contributed_at) }}</small>
            </span>
          </li>
        </ul>

        <div v-if="totalPage > 1" class="contributor-pagination">
          <KunPagination
            :current-page="page"
            :total-page="totalPage"
            :is-loading="loading"
            @update:current-page="changePage"
          />
        </div>
      </a-spin>
    </a-modal>
  </KunCard>
</template>

<style scoped>
.contributors-card {
  margin-bottom: 18px;
}

.contributors-heading {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.section-heading {
  margin-bottom: 4px;
}

.view-all {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 0;
  background: transparent;
  color: var(--color-primary);
  cursor: pointer;
  font-size: 14px;
}

.contributor-list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
}

.contributor-chip {
  max-width: 180px;
  padding: 7px 10px 7px 7px;
  border: 1px solid var(--color-default-200);
  border-radius: 999px;
}

.contributor-chip span:last-child {
  overflow: hidden;
  font-size: 13px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.contributor-tooltip {
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.contributor-modal-list {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.contributor-modal-list li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px;
  border: 1px solid var(--color-default-200);
  border-radius: var(--radius-kun-lg);
}

.contributor-identity span,
.contributor-stats {
  display: flex;
  min-width: 0;
  flex-direction: column;
}

.contributor-identity small,
.contributor-stats small {
  color: var(--color-default-500);
  font-size: 12px;
}

.contributor-stats {
  flex: 0 0 auto;
  align-items: flex-end;
  font-size: 13px;
}

.contributor-pagination {
  display: flex;
  justify-content: center;
  margin-top: 18px;
}

@media (max-width: 560px) {
  .contributor-modal-list li {
    align-items: flex-start;
    flex-direction: column;
  }

  .contributor-stats {
    align-items: flex-start;
    padding-left: 48px;
  }
}
</style>
