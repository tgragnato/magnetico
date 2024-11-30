package opflags

import (
	"os"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

func (o *OpFlags) Parse() (err error) {
	parser := flags.NewParser(o, flags.Default)
	_, err = parser.Parse()
	if err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		}
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
