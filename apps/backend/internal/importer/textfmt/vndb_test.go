package textfmt

import (
	"strings"
	"testing"
)

func TestVNDBMarkdownConversion(t *testing.T) {
	converter := NewVNDBMarkdownConverter()
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "internal character link",
			input: "[url=/c153631]Okinami Yukitaka[/url]",
			want:  "[Okinami Yukitaka](https://vndb.org/c153631)",
		},
		{
			name:  "internal vn link",
			input: "[url=/v17]Ever17[/url]",
			want:  "[Ever17](https://vndb.org/v17)",
		},
		{
			name:  "external link",
			input: "[url=https://example.com]Website[/url]",
			want:  "[Website](https://example.com)",
		},
		{
			name:  "bold",
			input: "[b]Hello[/b]",
			want:  "**Hello**",
		},
		{
			name:  "italic",
			input: "[i]Hello[/i]",
			want:  "*Hello*",
		},
		{
			name:  "underline keeps text",
			input: "[u]Hello[/u]",
			want:  "Hello",
		},
		{
			name:  "strikethrough",
			input: "[s]Hello[/s]",
			want:  "~~Hello~~",
		},
		{
			name:  "nested bold link",
			input: "[b][url=/c123]Alice[/url][/b]",
			want:  "**[Alice](https://vndb.org/c123)**",
		},
		{
			name:  "link with bare target",
			input: "[url]https://example.com[/url]",
			want:  "[https://example.com](https://example.com)",
		},
		{
			name:  "multi paragraph",
			input: "Hello.\n\n[url=/c123]Alice[/url] appears.",
			want:  "Hello.\n\n[Alice](https://vndb.org/c123) appears.",
		},
		{
			name:  "incomplete tag never panics",
			input: "[b]Hello",
			want:  "Hello",
		},
		{
			name:  "stray closing tag is dropped",
			input: "text[/b]",
			want:  "text",
		},
		{
			name:  "javascript url degrades to label",
			input: "[url=javascript:alert(1)]click[/url]",
			want:  "click",
		},
		{
			name:  "data url degrades to label",
			input: "[url=data:text/html;base64,PHNjcmlwdD4=]click[/url]",
			want:  "click",
		},
		{
			name:  "unknown tag strips codes and keeps content",
			input: "before [abc]hello[/abc] after",
			want:  "before hello after",
		},
		{
			name:  "unknown unterminated tag keeps content",
			input: "before [abc]hello",
			want:  "before hello",
		},
		{
			name:  "malformed bracket stays literal",
			input: "a [b b] c",
			want:  "a [b b] c",
		},
		{
			name:  "spoiler becomes spoiler blockquote",
			input: "She is [spoiler]the killer[/spoiler].",
			want:  "She is > **Spoiler:** the killer.",
		},
		{
			name:  "spoiler keeps multiline quoting",
			input: "[spoiler]first line\n\nsecond line[/spoiler]",
			want:  "> **Spoiler:** first line\n>\n> second line",
		},
		{
			name:  "quote prefixes every line",
			input: "[quote]first line\nsecond line[/quote]",
			want:  "> first line\n> second line",
		},
		{
			name:  "link label escaping",
			input: "[url=/c123]Alice [Route][/url]",
			want:  "[Alice \\[Route\\]](https://vndb.org/c123)",
		},
		{
			name:  "mismatched nesting keeps inner text",
			input: "[b]x[i]y[/b]",
			want:  "**xy**",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "empty tag renders nothing",
			input: "[b][/b] and [i][/i]",
			want:  " and ",
		},
		{
			name:  "tag names are case insensitive",
			input: "[B]Hello[/b]",
			want:  "**Hello**",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := converter.Convert(tc.input)
			if got != tc.want {
				t.Errorf("Convert(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestVNDBMarkdownAcceptanceSample(t *testing.T) {
	input := "Okinami meets [url=/c153631]Okinami Yukitaka[/url].\n\n[b]This section is important.[/b]"
	want := "Okinami meets [Okinami Yukitaka](https://vndb.org/c153631).\n\n**This section is important.**"
	got := NewVNDBMarkdownConverter().Convert(input)
	if got != want {
		t.Errorf("Convert acceptance sample = %q, want %q", got, want)
	}
}

func TestVNDBMarkdownCustomBaseURL(t *testing.T) {
	converter := &VNDBMarkdownConverter{BaseURL: "https://vndb.example.org/"}
	got := converter.Convert("[url=/v17]Ever17[/url]")
	want := "[Ever17](https://vndb.example.org/v17)"
	if got != want {
		t.Errorf("Convert = %q, want %q", got, want)
	}
}

func TestVNDBMarkdownRelativeLinkWithoutBaseURLStaysPlain(t *testing.T) {
	converter := &VNDBMarkdownConverter{}
	got := converter.Convert("[url=/v17]Ever17[/url]")
	if got != "Ever17" {
		t.Errorf("Convert = %q, want plain label %q", got, "Ever17")
	}
}

// TestVNDBMarkdownDoesNotEmitDangerousSchemes guards every URL-shaped input:
// conversion output must never contain a link to a non-http(s) scheme.
func TestVNDBMarkdownDoesNotEmitDangerousSchemes(t *testing.T) {
	converter := NewVNDBMarkdownConverter()
	dangerous := []string{
		"[url=javascript:alert(1)]x[/url]",
		"[url=vbscript:msgbox(1)]x[/url]",
		"[url=data:text/html;base64,PHNjcmlwdD4=]x[/url]",
		"[url=file:///etc/passwd]x[/url]",
		"[url=ftp://example.com/file]x[/url]",
		"[url=//example.com/evil]x[/url]",
		"[url=example.com/no-scheme]x[/url]",
		"[url=https://example.com/path with spaces]x[/url]",
		"[url=]x[/url]",
	}
	for _, input := range dangerous {
		output := converter.Convert(input)
		if strings.Contains(output, "](") {
			t.Errorf("Convert(%q) = %q, must not contain a markdown link", input, output)
		}
	}
}
