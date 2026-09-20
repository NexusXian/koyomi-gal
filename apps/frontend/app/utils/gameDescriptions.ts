import type { GameDescription, GameDescriptions } from '~/types/galgame'

export const DESCRIPTION_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const
export type DescriptionLocale = (typeof DESCRIPTION_LOCALES)[number]

// The site UI is Chinese-only today; the description viewer follows this
// locale until site-wide i18n exists or a manual switcher is added.
export const SITE_DESCRIPTION_LOCALE: DescriptionLocale = 'zh-CN'

const FALLBACK_CHAINS: Record<DescriptionLocale, DescriptionLocale[]> = {
  'zh-CN': ['zh-CN', 'ja-JP', 'en-US'],
  'ja-JP': ['ja-JP', 'zh-CN', 'en-US'],
  'en-US': ['en-US', 'ja-JP', 'zh-CN']
}

export const LOCALE_LABELS: Record<string, string> = {
  'zh-CN': '中文',
  'en-US': '英文',
  'ja-JP': '日文'
}

export function normalizeDescriptionLocale(
  locale: string | undefined | null
): DescriptionLocale {
  switch (locale) {
    case 'zh':
    case 'zh-CN':
    case 'zh-cn':
    case 'zh_CN':
    case 'zh-Hans':
      return 'zh-CN'
    case 'ja':
    case 'ja-JP':
    case 'ja-jp':
    case 'ja_JP':
      return 'ja-JP'
    default:
      return 'en-US'
  }
}

export interface BestDescription {
  description: GameDescription
  locale: DescriptionLocale
  isFallback: boolean
}

// Returns the description for the locale, falling back through the shared
// chain and skipping languages without usable content.
export function getBestDescription(
  descriptions: GameDescriptions | undefined | null,
  locale: string | undefined | null
): BestDescription | null {
  if (!descriptions) {
    return null
  }
  const normalized = normalizeDescriptionLocale(locale)
  for (const language of FALLBACK_CHAINS[normalized]) {
    const candidate = descriptions[language]
    if (candidate && candidate.content.trim() !== '') {
      return {
        description: candidate,
        locale: language,
        isFallback: language !== normalized
      }
    }
  }
  return null
}
