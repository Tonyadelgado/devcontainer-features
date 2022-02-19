package common

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var cachedContainerImageBuildMode = ""

// Required configuration for processing
type DevpackSettings struct {
	Publisher  string   // aka GitHub Org
	FeatureSet string   // aka GitHub Repository
	Version    string   // Used for version pinning
	ApiVersion string   // Buildpack API version to target
	Stacks     []string // Array of stacks that the buildpack should support

	//func (dp *DevpackSettings) Load(featuresPath string)
}

func (dp *DevpackSettings) Load(featuresPath string) {
	if featuresPath == "" {
		featuresPath = os.Getenv(BuildpackDirEnvVar)
	}
	content, err := ioutil.ReadFile(filepath.Join(featuresPath, DevpackSettingsFilename))
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(content, dp)
	if err != nil {
		log.Fatal(err)
	}
}

func GetContainerImageBuildMode() string {
	if cachedContainerImageBuildMode != "" {
		return cachedContainerImageBuildMode
	}
	cachedContainerImageBuildMode := os.Getenv(ContainerImageBuildModeEnvVarName)
	if cachedContainerImageBuildMode == "" {
		if _, err := os.Stat(ContainerImageBuildMarkerPath); err != nil {
			cachedContainerImageBuildMode = DefaultContainerImageBuildMode
		} else {
			fileBytes, err := os.ReadFile(ContainerImageBuildMarkerPath)
			if err != nil {
				cachedContainerImageBuildMode = DefaultContainerImageBuildMode
			} else {
				cachedContainerImageBuildMode = strings.TrimSpace(string(fileBytes))
			}
		}
	}
	return cachedContainerImageBuildMode
}
