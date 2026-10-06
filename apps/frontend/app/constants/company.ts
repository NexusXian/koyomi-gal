import type { DtoCompanyResponse, ListCompaniesParams } from '~/api/generated/models'

export const COMPANY_STATUSES: { value: NonNullable<DtoCompanyResponse['status']>; label: string }[] = [
  { value: 'active', label: '运营中' },
  { value: 'inactive', label: '停止活动' },
  { value: 'closed', label: '已关闭' }
]

export const COMPANY_SORTS: { value: NonNullable<ListCompaniesParams['sort']>; label: string }[] = [
  { value: 'created_at', label: '最新收录' },
  { value: 'name', label: '名称' },
  { value: 'game_count', label: '游戏数量' }
]

export const COMPANY_ROLES = [
  { value: 'developer', label: '开发商' },
  { value: 'publisher', label: '发行商' },
  { value: 'brand', label: '品牌' },
  { value: 'co_developer', label: '共同开发' }
] as const

export function companyRoleLabel(role?: string): string {
  return COMPANY_ROLES.find((item) => item.value === role)?.label ?? role ?? '公司'
}

export function companyStatusLabel(status?: string): string {
  return COMPANY_STATUSES.find((item) => item.value === status)?.label ?? status ?? '未知'
}
