package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsAndValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		yaml    string
		wantErr bool
	}{
		{"safe defaults", "directories: [data]\n", false},
		{"missing directory", "retention_months: 3\n", true},
		{"empty directories", "directories: []\n", true},
		{"blank directory", "directories: ['  ']\n", true},
		{"blank second directory", "directories: [data, '']\n", true},
		{"scalar directories", "directories: data\n", true},
		{"zero months", "directories: [data]\nretention_months: 0\n", true},
		{"negative months", "directories: [data]\nretention_months: -3\n", true},
		{"too many months", "directories: [data]\nretention_months: 1201\n", true},
		{"unknown field", "directories: [data]\ndry_rnu: false\n", true},
		{"duplicate field", "directories: [data]\ndry_run: true\ndry_run: false\n", true},
		{"invalid boolean", "directories: [data]\ndry_run: nope\n", true},
		{"malformed YAML", "directories: [\n", true},
		{"multiple documents", "directories: [data]\n---\ndry_run: false\n", true},
		{"empty document", "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			filename := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(filename, []byte(test.yaml), 0600); err != nil {
				t.Fatal(err)
			}
			settings, err := Load(filename)
			if (err != nil) != test.wantErr {
				t.Fatalf("Load error = %v, want error = %t", err, test.wantErr)
			}
			if !test.wantErr && (settings.RetentionMonths != 3 || !settings.DryRun || len(settings.Directories) != 1 || settings.Directories[0] != filepath.Join(filepath.Dir(filename), "data")) {
				t.Fatalf("incorrect defaults or relative path: %+v", settings)
			}
		})
	}
}
