import MarkdownIt from 'markdown-it'
import sanitizeHtml from 'sanitize-html'

// Single module-level parser instance: deterministic output for identical
// input on server and client, so SSR HTML matches hydration.
const md = new MarkdownIt({
  // Raw HTML stays disabled: user-supplied markup is escaped and rendered as
  // text, so scripts/event handlers can never reach the DOM from here.
  html: false,
  linkify: true,
  breaks: false,
  typographer: false
})

const SANITIZE_OPTIONS = {
  allowedTags: [
    'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'blockquote', 'ul', 'ol', 'li',
    'a', 'strong', 'em', 'del', 's',
    'code', 'pre', 'img', 'hr', 'br',
    'table', 'thead', 'tbody', 'tr', 'th', 'td'
  ],
  allowedAttributes: {
    a: ['href', 'title', 'target', 'rel'],
    img: ['src', 'alt', 'title'],
    th: ['style'],
    td: ['style']
  },
  allowedSchemes: ['http', 'https', 'mailto'],
  allowedSchemesByTag: {},
  allowProtocolRelative: false,
  // Only the table alignment styles markdown-it emits are permitted.
  allowedStyles: {
    th: { 'text-align': [/^(left|right|center|justify)$/] },
    td: { 'text-align': [/^(left|right|center|justify)$/] }
  },
  transformTags: {
    a: (tagName: string, attribs: Record<string, string>) => {
      if (!/^https?:\/\//i.test(attribs.href ?? '')) {
        return { tagName, attribs }
      }
      return {
        tagName,
        attribs: {
          ...attribs,
          target: '_blank',
          rel: 'noopener noreferrer nofollow ugc'
        }
      }
    }
  }
}

// Markdown -> HTML -> sanitized HTML. Never render the unsanitized output.
export function renderMarkdown(markdown: string): string {
  if (!markdown) {
    return ''
  }
  return sanitizeHtml(md.render(markdown), SANITIZE_OPTIONS)
}

// Lightweight markdown source stripping for plain-text excerpts (list cards,
// SEO descriptions). Intentionally regex-based; not a full parser.
export function stripMarkdownForExcerpt(markdown: string, maxLength = 200): string {
  if (!markdown) {
    return ''
  }
  const text = markdown
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`([^`]*)`/g, '$1')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/^\s{0,3}#{1,6}\s+/gm, '')
    .replace(/(\*{1,3}|_{1,3}|~~)(?=\S)([\s\S]*?\S)\1/g, '$2')
    .replace(/^\s*>\s?/gm, '')
    .replace(/^\s*[-+*]\s+/gm, '')
    .replace(/^\s*\d+[.)]\s+/gm, '')
    .replace(/^\s*\|?[\s:|-]+\|?\s*$/gm, ' ')
    .replace(/\|/g, ' ')
    .replace(/<[^>]+>/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  return text.length > maxLength ? `${text.slice(0, maxLength)}…` : text
}
