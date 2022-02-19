package common

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/tailscale/hujson"
)

// Pull in json as a simple map of maps given the structure
type DevContainerJson struct {
	Features map[string]interface{}
}

func loadDevContainerJsonConent(applicationFolder string) ([]byte, string) {
	devContainerJsonPath := FindDevContainerJson(applicationFolder)
	if devContainerJsonPath == "" {
		return []byte{}, devContainerJsonPath
	}
	content, err := ioutil.ReadFile(devContainerJsonPath)
	if err != nil {
		log.Fatal(err)
	}
	// Strip out comments to enable parsing
	ast, err := hujson.Parse(content)
	if err != nil {
		log.Fatal(err)
	}
	ast.Standardize()
	content = ast.Pack()

	return content, devContainerJsonPath
}

func (devContainerJson *DevContainerJson) Load(applicationFolder string) string {
	content, devContainerJsonPath := loadDevContainerJsonConent(applicationFolder)
	if devContainerJsonPath != "" {
		err := json.Unmarshal(content, devContainerJson)
		if err != nil {
			log.Fatal(err)
		}

	}
	return devContainerJsonPath
}

func LoadDevContainerJsonAsMap(applicationFolder string) (map[string]json.RawMessage, string) {
	jsonMap := make(map[string]json.RawMessage)
	content, devContainerJsonPath := loadDevContainerJsonConent(applicationFolder)
	if devContainerJsonPath != "" {
		err := json.Unmarshal(content, &jsonMap)
		if err != nil {
			log.Fatal(err)
		}
	}
	return jsonMap, devContainerJsonPath
}

func FindDevContainerJson(applicationFolder string) string {
	// Load devcontainer.json
	if applicationFolder == "" {
		var err error
		applicationFolder, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}

	expectedPath := filepath.Join(applicationFolder, ".devcontainer", "devcontainer.json")
	if _, err := os.Stat(expectedPath); err != nil {
		// if file does not exist, try .devcontainer.json instead
		if os.IsNotExist(err) {
			expectedPath = filepath.Join(applicationFolder, ".devcontainer.json")
			if _, err := os.Stat(expectedPath); err != nil {
				if !os.IsNotExist(err) {
					log.Fatal(err)
				}
				return ""
			}
		} else {
			log.Fatal(err)
		}
	}
	return expectedPath
}
