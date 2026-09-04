package textfmt

import (
	"net/url"
	"strings"
)

const defaultVNDBBaseURL = "https://vndb.org"

// VNDBMarkdownConverter converts VNDB formatting codes into Markdown.
//
// VNDB descriptions and other text fields may contain a BBCode subset
// ([url], [b], [i], [u], [s], [spoiler], [quote], ...). Because the site
// renders Markdown only, the codes are translated once at the VNDB boundary;
// the raw payload stays untouched in the provider snapshot.
type VNDBMarkdownConverter struct {
	// BaseURL prefixes VNDB relative links such as /v123.
	BaseURL string
}

// NewVNDBMarkdownConverter returns a converter for the public VNDB site.
func NewVNDBMarkdownConverter() *VNDBMarkdownConverter {
	return &VNDBMarkdownConverter{BaseURL: defaultVNDBBaseURL}
}

// Convert translates VNDB formatting codes into Markdown. Unknown and
// malformed codes degrade to their inner text so no source content is lost,
// and conversion never fails: external data is untrusted input.
func (c *VNDBMarkdownConverter) Convert(input string) string {
	return c.renderNodes(parseFormatting(input))
}

type node interface{}

type textNode struct {
	text string
}

type tagNode struct {
	name     string
	argument string
	// raw keeps the original opening text (e.g. "Route" or "b=2") for
	// literal replay when an unknown tag is left open by mismatched markup.
	raw      string
	children []node
}

// parseFormatting tokenizes a BBCode-style input into a tree of text and tag
// nodes. Mismatched or unterminated tags are tolerated: frames are popped
// until the matching opener is found (inner content stays attached to the
// closer frame) and stray closers are dropped, so well-formed markup converts
// cleanly and malformed input never panics or loses content.
func parseFormatting(input string) []node {
	root := &tagNode{}
	stack := []*tagNode{root}
	var text strings.Builder
	appendText := func() {
		if text.Len() == 0 {
			return
		}
		current := stack[len(stack)-1]
		current.children = append(current.children, textNode{text: text.String()})
		text.Reset()
	}

	i := 0
	for i < len(input) {
		open := strings.IndexByte(input[i:], '[')
		if open < 0 {
			text.WriteString(input[i:])
			break
		}
		open += i
		closeBracket := strings.IndexByte(input[open+1:], ']')
		if closeBracket < 0 {
			text.WriteString(input[i:])
			break
		}
		closeBracket += open + 1

		raw := input[open+1 : closeBracket]
		name, argument, ok := parseTagSpec(raw)
		text.WriteString(input[i:open])
		i = closeBracket + 1
		if !ok {
			// Malformed bracket: keep it as literal text.
			text.WriteString("[" + raw + "]")
			continue
		}
		appendText()
		if strings.HasPrefix(raw, "/") {
			stack = closeTag(stack, name)
			continue
		}
		child := &tagNode{name: name, argument: argument, raw: raw}
		stack[len(stack)-1].children = append(stack[len(stack)-1].children, child)
		stack = append(stack, child)
	}
	appendText()
	// Unterminated tags at end of input degrade to their plain content.
	for len(stack) > 1 {
		top := stack[len(stack)-1]
		parent := stack[len(stack)-2]
		parent.children = append(parent.children[:len(parent.children)-1], top.children...)
		stack = stack[:len(stack)-1]
	}
	return root.children
}

// parseTagSpec validates and splits "[name]", "[name=argument]" and
// "[/name]" contents. Malformed specs return ok=false and stay literal.
func parseTagSpec(raw string) (name, argument string, ok bool) {
	if raw == "" {
		return "", "", false
	}
	rest := raw
	if strings.HasPrefix(rest, "/") {
		rest = rest[1:]
		if rest == "" || strings.Contains(rest, "=") {
			return "", "", false
		}
		return strings.ToLower(rest), "", validTagName(rest)
	}
	if at := strings.IndexByte(rest, '='); at >= 0 {
		name = rest[:at]
		argument = rest[at+1:]
	} else {
		name = rest
	}
	if !validTagName(name) {
		return "", "", false
	}
	return strings.ToLower(name), argument, true
}

func validTagName(value string) bool {
	if value == "" {
		return false
	}
	for i := 0; i < len(value); i++ {
		char := value[i]
		isLetter := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
		isNumber := char >= '0' && char <= '9'
		if !isLetter && !isNumber && char != '_' && char != '-' {
			return false
		}
	}
	return true
}

// knownTagNames are the tags VNDB descriptions use. Tags outside this set
// are stripped on well-formed closes and rendered literally when they are
// left open by a mismatched close, so stray brackets never eat text.
var knownTagNames = map[string]bool{
	"url": true, "b": true, "i": true, "u": true, "s": true,
	"spoiler": true, "quote": true,
}

// closeTag pops frames until the tag named name is closed and returns the
// remaining stack. Inner frames above the match close implicitly: their
// content is spliced back into the matched frame so nothing is lost (unknown
// frames keep their literal opening code). A stray closer without any
// matching opener is dropped.
func closeTag(stack []*tagNode, name string) []*tagNode {
	match := -1
	for index := len(stack) - 1; index >= 1; index-- {
		if stack[index].name == name {
			match = index
			break
		}
	}
	if match < 0 {
		return stack
	}
	for index := len(stack) - 1; index > match; index-- {
		inner := stack[index]
		parent := stack[index-1]
		children := inner.children
		if !knownTagNames[inner.name] {
			children = append([]node{textNode{text: "[" + inner.raw + "]"}}, children...)
		}
		parent.children = append(parent.children[:len(parent.children)-1], children...)
	}
	return stack[:match]
}

// renderNodes renders parsed nodes into Markdown.
func (c *VNDBMarkdownConverter) renderNodes(nodes []node) string {
	var out strings.Builder
	c.writeNodes(nodes, &out)
	return out.String()
}

func (c *VNDBMarkdownConverter) writeNodes(nodes []node, out *strings.Builder) {
	for _, item := range nodes {
		switch typed := item.(type) {
		case textNode:
			out.WriteString(typed.text)
		case *tagNode:
			c.writeTag(typed, out)
		}
	}
}

func (c *VNDBMarkdownConverter) childrenText(tag *tagNode) string {
	var out strings.Builder
	c.writeNodes(tag.children, &out)
	return out.String()
}

func (c *VNDBMarkdownConverter) writeTag(tag *tagNode, out *strings.Builder) {
	switch tag.name {
	case "b", "i":
		marker := "*"
		if tag.name == "b" {
			marker = "**"
		}
		if inner := c.childrenText(tag); inner != "" {
			out.WriteString(marker)
			out.WriteString(inner)
			out.WriteString(marker)
		}
	case "s":
		if inner := c.childrenText(tag); inner != "" {
			out.WriteString("~~")
			out.WriteString(inner)
			out.WriteString("~~")
		}
	case "u":
		out.WriteString(c.childrenText(tag))
	case "url":
		out.WriteString(c.renderURL(tag))
	case "spoiler":
		out.WriteString(c.renderBlockquote("**Spoiler:** ", tag))
	case "quote":
		out.WriteString(c.renderBlockquote("", tag))
	default:
		// Unknown tag: strip the codes and keep the content so readers never
		// see raw BBCode.
		out.WriteString(c.childrenText(tag))
	}
}

// renderURL turns a [url=...] or [url]...[/url] tag into a Markdown link.
// Unsafe or unparseable targets degrade to plain link text; relative VNDB
// links are absolutized against BaseURL so they never resolve inside the
// site's own domain.
func (c *VNDBMarkdownConverter) renderURL(tag *tagNode) string {
	targetRaw := strings.TrimSpace(tag.argument)
	label := plainText(tag.children)
	labelFromContent := label != ""
	if !labelFromContent {
		label = targetRaw
	}
	if targetRaw == "" && labelFromContent {
		targetRaw = label
	}

	target := c.normalizeURL(targetRaw)
	if label == "" {
		label = target
	}
	if label == "" {
		return ""
	}
	if target == "" {
		if labelFromContent {
			return escapeLinkText(label)
		}
		return ""
	}
	return "[" + escapeLinkText(label) + "](" + target + ")"
}

// plainText flattens children into display text: tags are stripped and
// whitespace runs are collapsed. Used for link labels.
func plainText(nodes []node) string {
	var out strings.Builder
	writePlain(nodes, &out)
	return strings.Join(strings.Fields(out.String()), " ")
}

func writePlain(nodes []node, out *strings.Builder) {
	for _, item := range nodes {
		switch typed := item.(type) {
		case textNode:
			out.WriteString(typed.text)
		case *tagNode:
			writePlain(typed.children, out)
		}
	}
}

// normalizeURL validates a link target. Only absolute http(s) URLs and
// absolute VNDB paths (/v123, /c153631, ...) are accepted; anything else
// (javascript:, data:, protocol-relative //host, scheme-less text, ...) is
// rejected so no dangerous or broken link is ever emitted.
func (c *VNDBMarkdownConverter) normalizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.ContainsAny(raw, " \t\r\n") {
		return ""
	}
	if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
		if base == "" {
			return ""
		}
		return base + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	return raw
}

// escapeLinkText escapes the Markdown punctuation that would otherwise break
// a link label: backslashes, opening and closing brackets.
func escapeLinkText(value string) string {
	return strings.NewReplacer(`\`, `\\`, `[`, `\[`, `]`, `\]`).Replace(value)
}

// renderBlockquote renders [quote] and [spoiler] content as a Markdown
// blockquote, prefixing every line so multi-line content stays quoted.
func (c *VNDBMarkdownConverter) renderBlockquote(prefix string, tag *tagNode) string {
	inner := strings.Trim(c.childrenText(tag), "\r\n")
	if inner == "" {
		return ""
	}
	lines := strings.Split(inner, "\n")
	var out strings.Builder
	for index, line := range lines {
		line = strings.TrimRight(line, "\r")
		if index == 0 {
			out.WriteString("> ")
			if prefix != "" {
				out.WriteString(prefix)
			}
			out.WriteString(line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			out.WriteString("\n>")
		} else {
			out.WriteString("\n> ")
			out.WriteString(line)
		}
	}
	return out.String()
}
