package i18n_test

// Guards over the repo's actual locales/ bundle.
//
// These exist because a locale can be broken in a way nothing else notices: a
// locale file that is a byte-for-byte copy of the English strings passes a key
// parity check while serving English — you have to assert the values actually
// differ.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// shippingLocales must stay in full lockstep with en — same keys, real
// translations. This repo ships en as the only full-parity bundle today; id is
// a partial bundle (see partialLocales) that falls back to en per key.
var shippingLocales = []string{}

// partialLocales may omit sections (the Translator falls back to en per key),
// but whatever they DO define must be a genuine translation, not an English copy.
var partialLocales = []string{"id"}

// untranslatedByDesign are keys whose value is legitimately identical across
// every language. Anything not listed here must differ from en.
var untranslatedByDesign = map[string]string{
	"email.layout.brandName":  "product name — brands are not translated",
	"email.welcome.feat1Icon": "emoji — no language",
	"email.welcome.feat2Icon": "emoji — no language",
	"email.welcome.feat3Icon": "emoji — no language",
}

// Both placeholder dialects in play: Go templates ({{.Field}}) for API messages,
// and single braces ({name}) for the email templates.
var placeholderRe = regexp.MustCompile(`\{\{\.\w+\}\}|\{\w+\}`)

func localesDir() string { return filepath.Join("..", "..", "locales") }

// loadFlat reads a locale file into dot-separated key → value pairs.
func loadFlat(t *testing.T, lang string) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(localesDir(), lang+".json"))
	if err != nil {
		t.Fatalf("read %s locale: %v", lang, err)
	}
	var doc map[string]any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("parse %s locale: %v", lang, err)
	}
	out := map[string]string{}
	var walk func(prefix string, m map[string]any)
	walk = func(prefix string, m map[string]any) {
		for k, v := range m {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			switch val := v.(type) {
			case string:
				out[key] = val
			case map[string]any:
				walk(key, val)
			default:
				t.Errorf("%s: %s is neither a string nor an object (%T)", lang, key, v)
			}
		}
	}
	walk("", doc)
	return out
}

func placeholders(s string) []string {
	found := placeholderRe.FindAllString(s, -1)
	sort.Strings(found)
	return found
}

func allNonEnglishLocales() []string {
	return append(append([]string{}, shippingLocales...), partialLocales...)
}

func TestLocales_ShippingLocalesMatchEnglishKeySet(t *testing.T) {
	en := loadFlat(t, "en")
	for _, lang := range shippingLocales {
		other := loadFlat(t, lang)
		for key := range en {
			if _, ok := other[key]; !ok {
				t.Errorf("%s is missing key %s", lang, key)
			}
		}
		for key := range other {
			if _, ok := en[key]; !ok {
				t.Errorf("%s has key %s that en does not define", lang, key)
			}
		}
	}
}

func TestLocales_PartialLocalesDefineNoUnknownKeys(t *testing.T) {
	en := loadFlat(t, "en")
	for _, lang := range partialLocales {
		for key := range loadFlat(t, lang) {
			if _, ok := en[key]; !ok {
				t.Errorf("%s has key %s that en does not define", lang, key)
			}
		}
	}
}

// The regression guard: a locale that repeats the English string is not
// translated, however healthy the key set looks.
func TestLocales_NoValueIsAVerbatimEnglishCopy(t *testing.T) {
	en := loadFlat(t, "en")
	for _, lang := range allNonEnglishLocales() {
		other := loadFlat(t, lang)
		var offenders []string
		for key, val := range other {
			enVal, ok := en[key]
			if !ok || val != enVal {
				continue
			}
			if _, allowed := untranslatedByDesign[key]; allowed {
				continue
			}
			offenders = append(offenders, key+" = "+val)
		}
		sort.Strings(offenders)
		if len(offenders) > 0 {
			t.Errorf("%s has %d untranslated (English) values:\n  %s",
				lang, len(offenders), strings.Join(offenders, "\n  "))
		}
	}
}

// A translation that drops, renames or mistypes a placeholder silently loses the
// value at runtime — Go's text/template renders the rest and moves on.
func TestLocales_PlaceholdersSurviveTranslation(t *testing.T) {
	en := loadFlat(t, "en")
	for _, lang := range allNonEnglishLocales() {
		for key, val := range loadFlat(t, lang) {
			enVal, ok := en[key]
			if !ok {
				continue
			}
			want, got := placeholders(enVal), placeholders(val)
			if strings.Join(want, ",") != strings.Join(got, ",") {
				t.Errorf("%s: %s placeholder mismatch\n  en = %q %v\n  %s = %q %v",
					lang, key, enVal, want, lang, val, got)
			}
		}
	}
}

// The allowlist is a standing exemption, so it must not outlive its reason.
func TestLocales_UntranslatedAllowlistIsCurrent(t *testing.T) {
	en := loadFlat(t, "en")
	for key := range untranslatedByDesign {
		if _, ok := en[key]; !ok {
			t.Errorf("untranslatedByDesign lists %s, which en no longer defines — drop it", key)
		}
	}
}
