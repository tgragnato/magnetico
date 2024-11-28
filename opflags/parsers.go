package opflags

import (
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

func (o *OpFlags) Parse() (err error) {
	_, err = flags.Parse(o)
	if err != nil {
		return err
	}

	err = o.parseYaml()
	if err != nil {
		return err
	}

	return o.check()
}

func (o *OpFlags) parseYaml() error {
	if o.ConfigFilePath == "" {
		return nil
	}

	data, err := os.ReadFile(o.ConfigFilePath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, o)
}
