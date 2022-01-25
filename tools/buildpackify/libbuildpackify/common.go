package libbuildpackify

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type FeatureMount struct {
	Source string
	Target string
	Type   string
}

type FeatureConfig struct {
	Id           string
	Name         string
	Options      map[string]string
	Entrypoint   string
	Privileged   bool
	Init         bool
	ContainerEnv []string
	Mounts       []FeatureMount
	CapAdd       []string
	SecurityOpt  []string
	BuildArg     string
}

type FeaturesJson struct {
	Features []FeatureConfig
}

// Required manual configuration
type BuildpackSettings struct {
	Publisher  string // aka GitHub Org
	FeatureSet string // aka GitHub Repository
}

// Pull in json as a simple map of maps given the structure
type DevContainerJson struct {
	Features map[string]map[string]string
}

func LoadFeaturesJson() FeaturesJson {
	// Load features.json
	content, err := ioutil.ReadFile(filepath.Join(os.Getenv("CNB_BUILDPACK_DIR"), "features", "features.json"))
	if err != nil {
		log.Fatal(err)
	}
	var featuresJson FeaturesJson
	err = json.Unmarshal(content, &featuresJson)
	if err != nil {
		log.Fatal(err)
	}

	return featuresJson
}

func LoadBuildpackSettings() BuildpackSettings {
	// Load features.json
	content, err := ioutil.ReadFile(filepath.Join(os.Getenv("CNB_BUILDPACK_DIR"), "buildpack-settings.json"))
	if err != nil {
		log.Fatal(err)
	}
	var jsonContents BuildpackSettings
	err = json.Unmarshal(content, &jsonContents)
	if err != nil {
		log.Fatal(err)
	}

	return jsonContents
}
