export type AnnouncementType =
  | 'normal'
  | 'update'
  | 'maintenance'
  | 'warning'
  | 'event'
  | 'system'

export type AnnouncementDisplayMode =
  | 'normal'
  | 'banner'
  | 'modal'
  | 'startup_modal'

export type AnnouncementTarget = 'all' | 'web' | 'android' | 'ios' | 'desktop'

export interface Announcement {
  id: number
  title: string
  content: string
  type: AnnouncementType
  displayMode: AnnouncementDisplayMode
  target: AnnouncementTarget
  priority: number
  dismissible: boolean
  published?: boolean
  startsAt?: string | null
  endsAt?: string | null
  createdBy?: number | null
  createdAt?: string
  updatedAt?: string
}

export interface AnnouncementPayload {
  title: string
  content: string
  type: AnnouncementType
  displayMode: AnnouncementDisplayMode
  target: AnnouncementTarget
  priority: number
  dismissible: boolean
  published: boolean
  startsAt?: string | null
  endsAt?: string | null
}
