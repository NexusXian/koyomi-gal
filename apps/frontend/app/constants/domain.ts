export interface DomainOption {
  value: number
  label: string
  color?: string
  slug?: string
}

export const GALGAME_STATUS: DomainOption[] = [
  { value: 0, label: '待审核', color: 'default' },
  { value: 1, label: '已发布', color: 'success' },
  { value: 2, label: '已拒绝', color: 'error' },
  { value: 3, label: '已隐藏', color: 'warning' }
]

export const AGE_RATINGS: DomainOption[] = [
  { value: 0, label: '未分级' },
  { value: 1, label: '全年龄' },
  { value: 4, label: '12+' },
  { value: 2, label: '15+' },
  { value: 5, label: '17+' },
  { value: 3, label: '18+' }
]

export const USER_STATES: DomainOption[] = [
  { value: 1, label: '想玩' },
  { value: 2, label: '在玩' },
  { value: 3, label: '玩完' },
  { value: 4, label: '搁置' },
  { value: 5, label: '弃坑' }
]

export const RESOURCE_TYPES: DomainOption[] = [
  { value: 0, label: '其他' },
  { value: 1, label: '游戏本体' },
  { value: 2, label: '补丁' },
  { value: 3, label: '存档' },
  { value: 4, label: '原声集' },
  { value: 5, label: 'CG' },
  { value: 6, label: '攻略' },
  { value: 7, label: '官方网站' },
  { value: 8, label: '购买页面' },
  { value: 9, label: '电子书' },
  { value: 10, label: '实体书' },
  { value: 11, label: '翻译版本' },
  { value: 12, label: '资料合集' }
]

export const RESOURCE_STATUS: DomainOption[] = [
  { value: 0, label: '待审核', color: 'processing' },
  { value: 1, label: '已发布', color: 'success' },
  { value: 2, label: '已拒绝', color: 'error' },
  { value: 3, label: '已隐藏', color: 'warning' }
]

export const REPORT_REASONS: DomainOption[] = [
  { value: 0, label: '链接失效' },
  { value: 1, label: '密码错误' },
  { value: 2, label: '文件损坏' },
  { value: 3, label: '疑似恶意软件' },
  { value: 4, label: '版本不符' },
  { value: 5, label: '重复资源' },
  { value: 6, label: '其他' }
]

export const REPORT_STATUS: DomainOption[] = [
  { value: 0, label: '待处理', color: 'processing' },
  { value: 1, label: '已解决', color: 'success' },
  { value: 2, label: '已驳回', color: 'default' }
]

export const GALGAME_SORTS: DomainOption[] = [
  { value: 0, label: '最新', slug: 'latest' },
  { value: 1, label: '最早', slug: 'oldest' },
  { value: 2, label: '评分最高', slug: 'rating' },
  { value: 3, label: '收藏最多', slug: 'favorite' },
  { value: 4, label: '最热门', slug: 'popular' }
]

export const GALLERY_IMAGE_TYPES: DomainOption[] = [
  { value: 0, label: '游戏截图' },
  { value: 1, label: 'CG' },
  { value: 2, label: '角色' },
  { value: 3, label: 'UI' },
  { value: 4, label: '宣传图' }
]

export const GALLERY_IMAGE_STATUS: DomainOption[] = [
  { value: 0, label: '待审核', color: 'processing' },
  { value: 1, label: '已通过', color: 'success' },
  { value: 2, label: '已拒绝', color: 'error' }
]

export const GALLERY_SOURCE_TYPES: DomainOption[] = [
  { value: 0, label: '本站上传' },
  { value: 1, label: '外部链接' }
]

export const NOVEL_STATUS: DomainOption[] = [
  { value: 0, label: '待审核', color: 'default' },
  { value: 1, label: '已发布', color: 'success' },
  { value: 2, label: '已拒绝', color: 'error' },
  { value: 3, label: '已隐藏', color: 'warning' }
]

export const NOVEL_RELEASE_STATUS: DomainOption[] = [
  { value: 0, label: '连载中', slug: 'ongoing' },
  { value: 1, label: '已完结', slug: 'completed' },
  { value: 2, label: '休刊中', slug: 'hiatus' },
  { value: 3, label: '已放弃', slug: 'cancelled' },
  { value: 4, label: '未知', slug: 'unknown' }
]

export const NOVEL_SORTS: DomainOption[] = [
  { value: 0, label: '最近更新', slug: 'updated' },
  { value: 1, label: '最新收录', slug: 'latest' },
  { value: 2, label: '最早出版', slug: 'release_asc' },
  { value: 3, label: '最新出版', slug: 'release' }
]

export const NOVEL_RELATION_TYPES: DomainOption[] = [
  { value: 0, label: '改编作品', slug: 'adaptation' },
  { value: 1, label: '原作', slug: 'original' },
  { value: 2, label: '衍生作品', slug: 'spin_off' },
  { value: 3, label: '续作', slug: 'sequel' },
  { value: 4, label: '前作', slug: 'prequel' },
  { value: 5, label: '同系列', slug: 'same_series' },
  { value: 6, label: '相关作品', slug: 'related' }
]

export function domainSlug(options: DomainOption[], slug: string): DomainOption | undefined {
  return options.find((option) => option.slug === slug)
}

export function domainLabel(
  options: DomainOption[],
  value: number | undefined | null
): string {
  if (value === undefined || value === null) {
    return '-'
  }

  return options.find((option) => option.value === value)?.label ?? '-'
}

export function domainSortSlug(options: DomainOption[], value: number): string {
  return (
    (options.find((option) => option.value === value)?.slug as string) ??
    'latest'
  )
}

export function formatDate(value: string | undefined | null): string {
  if (!value) {
    return '-'
  }

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }

  return date.toLocaleString('zh-CN', { hour12: false })
}

export function formatPlayTime(minutes: number | undefined | null): string {
  if (!minutes || minutes <= 0) {
    return '0 分钟'
  }

  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60

  if (hours <= 0) {
    return `${rest} 分钟`
  }

  return rest > 0 ? `${hours} 小时 ${rest} 分钟` : `${hours} 小时`
}
