package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type NonZeroExitError struct {
	ExitCode int
}

func (err NonZeroExitError) Error() string {
	return "Non-zero exit code: " + strconv.FormatInt(int64(err.ExitCode), 10)
}

type FeatureMount struct {
	Source string
	Target string
	Type   string
}

type FeatureOption struct {
	Type        string
	Enum        []string
	Proposals   []string
	Default     string
	Description string
}

type FeatureConfig struct {
	Id           string
	Name         string
	Options      map[string]FeatureOption
	Entrypoint   string
	Privileged   bool
	Init         bool
	ContainerEnv map[string]string
	Mounts       []FeatureMount
	CapAdd       []string
	SecurityOpt  []string
	BuildArg     string
}

type FeaturesJson struct {
	Features []FeatureConfig
}

// Required configuration for processing
type BuildpackSettings struct {
	Publisher  string   // aka GitHub Org
	FeatureSet string   // aka GitHub Repository
	Version    string   // Used for version pinning
	ApiVersion string   // Buildpack API version to target
	Stacks     []string // Array of stacks that the buildpack should support
}

// Pull in json as a simple map of maps given the structure
type DevContainerJson struct {
	Features map[string]interface{}
}

func LoadFeaturesJson(featuresPath string) FeaturesJson {
	// Load devcontainer-features.json or features.json
	if featuresPath == "" {
		featuresPath = os.Getenv("CNB_BUILDPACK_DIR")
	}
	var content []byte
	var err error
	content, err = ioutil.ReadFile(filepath.Join(featuresPath, "devcontainer-features.json"))
	if err != nil {
		content, err = ioutil.ReadFile(filepath.Join(featuresPath, "features.json"))
		if err != nil {
			log.Fatal(err)
		}
	}
	var featuresJson FeaturesJson
	err = json.Unmarshal(content, &featuresJson)
	if err != nil {
		log.Fatal(err)
	}

	return featuresJson
}

func LoadBuildpackSettings(featuresPath string) BuildpackSettings {
	if featuresPath == "" {
		featuresPath = os.Getenv("CNB_BUILDPACK_DIR")
	}
	content, err := ioutil.ReadFile(filepath.Join(featuresPath, "buildpack-settings.json"))
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

func LoadDevContainerJson() DevContainerJson {
	// Load devcontainer.json
	var content []byte
	var err error
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	content, err = ioutil.ReadFile(filepath.Join(cwd, "devcontainer", "devcontainer.json"))
	if err != nil {
		content, err = ioutil.ReadFile(filepath.Join(cwd, ".devcontainer.json"))
		if err != nil {
			log.Fatal(err)
		}
	}
	var jsonContents DevContainerJson
	err = json.Unmarshal(content, &jsonContents)
	if err != nil {
		log.Fatal(err)
	}

	return jsonContents
}

func GetFeatureScriptPath(buidpackPath string, featureId string, script string) string {
	return filepath.Join(buidpackPath, "features", featureId, "bin", script)
}

func ContainerBuildContext() string {
	context := os.Getenv("BP_CONTAINER_FEATURE_BUILD_CONTEXT")
	if context == "" {
		return "devcontainer"
	}
	return context
}

func GetBuildEnvironment(feature FeatureConfig, targetLayerPath string) ([]string, map[string]string) {
	// Create environment that includes feature build args
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	optionEnvVarPrefix := "_BUILD_ARG_" + idSafe
	env := append(os.Environ(),
		optionEnvVarPrefix+"=true",
		"_FEATURE_BUILD_CONTEXT="+ContainerBuildContext())
	if targetLayerPath != "" {
		env = append(env, optionEnvVarPrefix+"_TARGET_PATH="+targetLayerPath)
	}

	setOptions := make(map[string]string)

	// TODO: Inspect devcontainer.json if present to find options

	// Look for BP_CONTAINER_FEATURE_<feature.Id>_<option> environment variables, convert
	for optionName := range feature.Options {
		optionNameSafe := strings.ReplaceAll(strings.ToUpper(optionName), "-", "_")
		optionValue := os.Getenv("BP_DEV_CONTAINER_FEATURE_" + idSafe + "_" + optionNameSafe)
		if optionValue != "" {
			env = append(env, optionEnvVarPrefix+"_"+optionName+"=\""+optionValue+"\"")
			setOptions[optionName] = optionValue
		}
	}
	return env, setOptions
}
