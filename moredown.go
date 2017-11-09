package moredown

import (
	"bytes"
	"io"
	"regexp"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/microcosm-cc/bluemonday"
	blackfriday "gopkg.in/russross/blackfriday.v1"
)

const blackfridayExtensions = blackfriday.EXTENSION_TABLES |
	blackfriday.EXTENSION_FENCED_CODE |
	blackfriday.EXTENSION_AUTOLINK |
	blackfriday.EXTENSION_STRIKETHROUGH |
	blackfriday.EXTENSION_SPACE_HEADERS |
	blackfriday.EXTENSION_NO_EMPTY_LINE_BEFORE_BLOCK

const htmlFlags = blackfriday.HTML_HREF_TARGET_BLANK

var markdownRenderer = &renderer{blackfriday.HtmlRenderer(htmlFlags, "", "").(*blackfriday.Html)}
var policy = newUGCPolicy()

func newUGCPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(bluemonday.SpaceSeparatedTokens).OnElements("pre", "span")
	p.AllowAttrs("class", "name").Matching(bluemonday.SpaceSeparatedTokens).OnElements("a")
	p.AllowAttrs("rel").Matching(regexp.MustCompile(`^nofollow$`)).OnElements("a")
	p.AllowAttrs("aria-hidden").Matching(regexp.MustCompile(`^true$`)).OnElements("a")
	return p
}

func MarkdownString(s string) string {
	return string(Markdown([]byte(s)))
}

func Markdown(b []byte) []byte {
	rendered := blackfriday.Markdown(b, markdownRenderer, blackfridayExtensions)
	sanitized := policy.SanitizeBytes(rendered)
	return sanitized
}

type renderer struct {
	*blackfriday.Html
}

func (r *renderer) BlockCode(out *bytes.Buffer, text []byte, lang string) {
	doubleSpace(out)

	if lang == "" {
		out.WriteString(`<pre>`)
	}

	text = bytes.TrimSpace(text)
	if lang == "" {
		escapeHTML(out, text)
	} else {
		err := writeHighlightedCode(out, string(text), lang)
		if err != nil {
			escapeHTML(out, text)
		}
	}

	if lang == "" {
		out.WriteString("</pre>\n")
	}
}

func doubleSpace(out *bytes.Buffer) {
	if out.Len() > 0 {
		out.WriteByte('\n')
	}
}

func escapeHTML(out *bytes.Buffer, src []byte) {
	last := 0

	for i, ch := range src {
		escaped, isEscaped := escapeChar(ch)
		if isEscaped {
			if i > last {
				out.Write(src[last:i])
			}
			out.WriteString(escaped)
			last = i + 1
		}
	}

	if last < len(src) {
		out.Write(src[last:])
	}
}

func escapeChar(char byte) (string, bool) {
	switch char {
	case '"':
		return "&quot;", true
	case '&':
		return "&amp;", true
	case '<':
		return "&lt;", true
	case '>':
		return "&gt;", true
	}
	return "", false
}

func writeHighlightedCode(dst io.Writer, src string, lang string) error {
	// Determine lexer.
	l := lexers.Get(lang)
	if l == nil {
		l = lexers.Analyse(src)
	}
	if l == nil {
		l = lexers.Fallback
	}
	l = chroma.Coalesce(l)

	// Determine formatter.
	f := html.New(html.WithClasses())

	// Determine style.
	s := styles.Get("abap")
	if s == nil {
		s = styles.Fallback
	}

	it, err := l.Tokenise(nil, src)
	if err != nil {
		return err
	}
	return f.Format(dst, s, it)
}
