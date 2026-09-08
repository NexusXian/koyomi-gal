export type ChangelogItemType = 'new' | 'improve' | 'fix'

export interface ChangelogItem {
  type: ChangelogItemType
  text: string
}

export interface SiteChangelog {
  id: number
  version: string
  title: string
  items: ChangelogItem[]
  publishedAt: string
  createdAt?: string
  updatedAt?: string
}

export interface SiteChangelogPayload {
  version: string
  title: string
  publishedAt?: string | null
  items: ChangelogItem[]
}
