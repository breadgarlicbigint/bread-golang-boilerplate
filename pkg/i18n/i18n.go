package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/gin-gonic/gin"
)

const (
	DefaultLang    = "en"
	LangHeader     = "x-custom-lang"
	CtxLang        = "lang"
	CtxTranslator  = "translator"
)

// Translator resolves i18n keys to localised strings.
type Translator struct {
	mu      sync.RWMutex
	locales map[string]map[string]interface{} // lang → nested key map
}

// New loads all *.json files from the given directory and returns a Translator.
func New(localesDir string) (*Translator, error) {
	t := &Translator{locales: make(map[string]map[string]interface{})}

	entries, err := os.ReadDir(localesDir)
	if err != nil {
		return nil, fmt.Errorf("i18n: read dir %q: %w", localesDir, err)
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}

		lang := strings.TrimSuffix(e.Name(), ".json")
		path := filepath.Join(localesDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("i18n: read %q: %w", path, err)
		}

		var m map[string]interface{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("i18n: parse %q: %w", path, err)
		}
		t.locales[lang] = m
	}

	if _, ok := t.locales[DefaultLang]; !ok {
		return nil, fmt.Errorf("i18n: default locale %q not found in %q", DefaultLang, localesDir)
	}
	return t, nil
}

// T translates a dot-separated key for the given language.
// Falls back to DefaultLang if the key is missing in the requested language.
// data is an optional map used for Go template interpolation ({{.Field}}).
func (t *Translator) T(lang, key string, data ...map[string]string) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	val := t.resolve(lang, key)
	if val == "" {
		val = t.resolve(DefaultLang, key)
	}
	if val == "" {
		return key // last resort: return the key itself
	}

	// Template interpolation
	if len(data) > 0 && len(data[0]) > 0 {
		tpl, err := template.New("").Parse(val)
		if err != nil {
			return val
		}
		var buf bytes.Buffer
		if err := tpl.Execute(&buf, data[0]); err == nil {
			val = buf.String()
		}
	}
	return val
}

// SupportedLanguages returns the list of loaded language codes.
func (t *Translator) SupportedLanguages() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	langs := make([]string, 0, len(t.locales))
	for k := range t.locales {
		langs = append(langs, k)
	}
	return langs
}

// resolve navigates the nested map using dot-separated keys.
func (t *Translator) resolve(lang, key string) string {
	m, ok := t.locales[lang]
	if !ok {
		return ""
	}
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 1 {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}
	sub, ok := m[parts[0]].(map[string]interface{})
	if !ok {
		return ""
	}
	return resolveNested(sub, parts[1])
}

func resolveNested(m map[string]interface{}, key string) string {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) == 1 {
		if v, ok := m[key].(string); ok {
			return v
		}
		return ""
	}
	sub, ok := m[parts[0]].(map[string]interface{})
	if !ok {
		return ""
	}
	return resolveNested(sub, parts[1])
}

// ── Gin middleware ────────────────────────────────────────────────────────────

// Middleware reads the x-custom-lang request header and stores the resolved
// lang + a bound translator in the Gin context.
func Middleware(t *Translator) gin.HandlerFunc {
	supported := make(map[string]bool)
	for _, l := range t.SupportedLanguages() {
		supported[l] = true
	}

	return func(c *gin.Context) {
		lang := strings.ToLower(strings.TrimSpace(c.GetHeader(LangHeader)))
		if lang == "" || !supported[lang] {
			lang = DefaultLang
		}
		c.Set(CtxLang, lang)
		c.Set(CtxTranslator, t)
		c.Next()
	}
}

// ── Gin context helpers ───────────────────────────────────────────────────────

// FromContext returns the Translator and active language from a Gin context.
func FromContext(c *gin.Context) (*Translator, string) {
	tr, _ := c.Get(CtxTranslator)
	lang, _ := c.Get(CtxLang)
	t, _ := tr.(*Translator)
	l, _ := lang.(string)
	if l == "" {
		l = DefaultLang
	}
	return t, l
}

// TC is a shorthand: translate a key using the language in the Gin context.
func TC(c *gin.Context, key string, data ...map[string]string) string {
	tr, lang := FromContext(c)
	if tr == nil {
		return key
	}
	return tr.T(lang, key, data...)
}
