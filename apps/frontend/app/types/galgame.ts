import type {
  DtoCreateGalgameRequest,
  DtoGalgameResponse,
  DtoUpdateGalgameRequest
} from '~/api/generated/models'

export interface DescriptionSource {
  type: string
  name: string
  url?: string
  official: boolean
}

export interface GameDescription {
  language: string
  content: string
  source: DescriptionSource
}

export type GameDescriptions = Record<string, GameDescription>

// Orval types lag behind the backend's descriptions field; detail responses
// are widened with it until the swagger endpoint is regenerated.
export type GalgameDetailData = DtoGalgameResponse & {
  descriptions?: GameDescriptions
}

export interface GalgameDescriptionPayloadItem {
  language: string
  content: string
  source_type: string
  source_name: string
  source_url: string
  is_official: boolean
}

// Update payloads carry the per-language descriptions until Orval regenerates.
export type GalgameUpdatePayload = DtoUpdateGalgameRequest & {
  descriptions?: GalgameDescriptionPayloadItem[]
}

export type GalgameCreatePayload = DtoCreateGalgameRequest & {
  descriptions?: GalgameDescriptionPayloadItem[]
}
