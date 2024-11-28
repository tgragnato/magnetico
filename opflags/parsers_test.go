package opflags

import (
	"testing"
)

func TestParseYaml(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		ConfigFilePath string
		expectError    bool
	}{{
		name:           "ValidConfigFilePath",
		ConfigFilePath: "../doc/config.example.yml",
		expectError:    false,
	}, {
		name:           "EmptyConfigFilePath",
		ConfigFilePath: "",
		expectError:    false,
	}, {
		name:           "InvalidConfigFilePath",
		ConfigFilePath: "invalid-path",
		expectError:    true,
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &OpFlags{
				ConfigFilePath: tt.ConfigFilePath,
			}
			if err := o.parseYaml(); (err != nil) != tt.expectError {
				t.Errorf("OpFlags.parseYaml() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
