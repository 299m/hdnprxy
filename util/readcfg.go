package util

import (
	"encoding/json"
	"os"
	filepath2 "path/filepath"
)

type Expandable interface {
	Expand()
}

func ReadConfig(basedir string, cfg map[string]Expandable) {
	// Read the config file
	// Unmarshal the config file into cfg
	// Return the cfg
	for file, something := range cfg {
		filepath := filepath2.Join(basedir, file+".json")
		filedata, err := os.ReadFile(filepath)
		CheckError(err)
		err = json.Unmarshal(filedata, something)
		CheckError(err)
		something.Expand()
	}
}
