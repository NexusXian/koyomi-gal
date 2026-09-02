export type FeedbackType = 'feedback' | 'copyright'

export interface FeedbackData {
  id: number
  type: FeedbackType
  content: string
  contact: string
  user_id: number | null
  ip: string
  user_agent: string
  handled_by: number | null
  handled_at: string | null
  created_at: string
  updated_at: string
}

export interface FeedbackPayload {
  type: FeedbackType
  content: string
  contact?: string
}
