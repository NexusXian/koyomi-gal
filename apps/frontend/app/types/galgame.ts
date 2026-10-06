import type {
  DtoCreateGalgameRequest,
  DtoDescriptionSourceResponse,
  DtoGalgameDescriptionInput,
  DtoGalgameDescriptionResponse,
  DtoGalgameResponse,
  DtoUpdateGalgameRequest
} from '~/api/generated/models'

export type DescriptionSource = DtoDescriptionSourceResponse
export type GameDescription = DtoGalgameDescriptionResponse
export type GameDescriptions = NonNullable<DtoGalgameResponse['descriptions']>
export type GalgameDetailData = DtoGalgameResponse
export type GalgameDescriptionPayloadItem = DtoGalgameDescriptionInput
export type GalgameUpdatePayload = DtoUpdateGalgameRequest
export type GalgameCreatePayload = DtoCreateGalgameRequest
