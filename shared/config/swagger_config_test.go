package config

import (
	"os"
	"path/filepath"
	"testing"
)

// An unset SWAGGER_ENABLED keeps the pre-flag behaviour: on outside production,
// off in it. Production turns it on only by saying so.
func TestSwaggerEnabledDefaultFollowsAppEnv(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.env")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("BREAD_CONFIG_FILE", empty)
	cases := []struct {
		env, flag string
		want      bool
	}{
		{"development", "", true},
		{"staging", "", true},
		{"production", "", false},
		{"production", "true", true},
		{"development", "false", false},
	}
	for _, tc := range cases {
		t.Run(tc.env+"/"+tc.flag, func(t *testing.T) {
			t.Setenv("APP_ENV", tc.env)
			t.Setenv("SWAGGER_ENABLED", tc.flag) // registers the restore
			if tc.flag == "" {
				os.Unsetenv("SWAGGER_ENABLED")
			}
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Swagger.Enabled != tc.want {
				t.Errorf("Swagger.Enabled = %v, want %v", cfg.Swagger.Enabled, tc.want)
			}
		})
	}
}
