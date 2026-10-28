package config

// Config hygiene: every env var the config binds (a `mapstructure` tag) must be
// documented in .env.example, so new config never ships undocumented. New
// config that is added to a struct but forgotten in .env.example is how an
// operator ends up unable to discover a knob that exists.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var mapstructureRe = regexp.MustCompile(`mapstructure:"([^"]+)"`)

// configEnvVars returns every env var named by a mapstructure tag across the
// config package source.
func configEnvVars(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read config dir: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile(e.Name())
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		for _, m := range mapstructureRe.FindAllStringSubmatch(string(b), -1) {
			if m[1] != "" && m[1] != "-" {
				out[m[1]] = e.Name()
			}
		}
	}
	if len(out) < 20 {
		t.Fatalf("only %d env vars found in config — the scan is probably broken", len(out))
	}
	return out
}

// envExampleKeys reads the KEY= lines of .env.example.
func envExampleKeys(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(filepath.Join("..", "..", ".env.example"))
	if err != nil {
		t.Fatalf("open .env.example: %v", err)
	}
	defer f.Close()
	keys := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		// A documented-but-commented default (e.g. "# SWAGGER_ENABLED=") still
		// counts as documented.
		line = strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if line == "" {
			continue
		}
		if k, _, ok := strings.Cut(line, "="); ok {
			keys[strings.TrimSpace(k)] = true
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	return keys
}

func TestEveryConfigEnvVarIsDocumented(t *testing.T) {
	vars := configEnvVars(t)
	documented := envExampleKeys(t)

	var missing []string
	for name, file := range vars {
		if !documented[name] {
			missing = append(missing, name+"  ("+file+")")
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf(".env.example is missing %d documented env var(s) the config binds:\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}
}
