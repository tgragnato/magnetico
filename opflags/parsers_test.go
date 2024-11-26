package opflags

import (
	"testing"
)

func TestParseYaml(t *testing.T) {
	t.Parallel()

	o := &OpFlags{
		ConfigFilePath: "../doc/config.example.yml",
	}

	if err := o.parseYaml(); err != nil {
		t.Fatal(err)
	}
}
