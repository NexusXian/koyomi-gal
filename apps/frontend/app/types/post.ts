export type EditorMode = 'plain' | 'markdown'

export const EDITOR_MODE_OPTIONS: { label: string; value: EditorMode }[] = [
  { label: '普通编辑', value: 'plain' },
  { label: 'Markdown', value: 'markdown' }
]

// API responses type editor_mode as a loose string; normalize unknown or
// missing values to plain so historical posts keep their old rendering.
export function normalizeEditorMode(value: unknown): EditorMode {
  return value === 'markdown' ? 'markdown' : 'plain'
}
