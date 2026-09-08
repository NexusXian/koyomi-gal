export type AppReleasePlatform =
  | 'android'
  | 'ios'
  | 'windows'
  | 'macos'
  | 'linux'

export type AppReleaseStatus = 'draft' | 'published' | 'disabled'

export interface AppRelease {
  id: number
  platform: AppReleasePlatform
  versionName: string
  versionCode: number
  title: string
  changelog: string
  downloadUrl: string
  fileSize?: number | null
  sha256?: string | null
  minimumVersionCode: number
  forceUpdate: boolean
  status: AppReleaseStatus
  publishedAt?: string | null
  announcementId?: number | null
  createdAt: string
  updatedAt: string
}

export interface AppReleasePayload {
  platform: AppReleasePlatform
  versionName: string
  versionCode: number
  title: string
  changelog: string
  downloadUrl: string
  fileSize?: number | null
  sha256?: string | null
  minimumVersionCode: number
  forceUpdate: boolean
  createAnnouncement: boolean
  status: AppReleaseStatus
  publishedAt?: string | null
  announcementId?: number | null
}

export interface GitHubReleaseAsset {
  name: string
  size: number
  sha256?: string | null
  downloadUrl: string
  updatedAt: string
}

export interface GitHubRelease {
  tag: string
  name: string
  body: string
  prerelease: boolean
  createdAt: string
  publishedAt: string
  apk?: GitHubReleaseAsset | null
}

export interface LatestAppRelease {
  hasUpdate: boolean
  forceUpdate?: boolean
  latestVersion?: { versionName: string; versionCode: number }
  title?: string
  changelog?: string
  downloadUrl?: string
  fileSize?: number | null
  sha256?: string | null
  publishedAt?: string
}
