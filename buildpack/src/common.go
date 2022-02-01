package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const DefaultApiVersion = "0.7"
const MetadataIdPrefix = "com.microsoft.devcontainer"
const FeaturesetMetadataId = "featureset"
const FeaturesMetadataId = "features"
const AppliedFeaturesLabelId = MetadataIdPrefix + ".features"

var devcontainerJsonCache map[string]DevContainerJson

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
	content, err := ioutil.ReadFile(filepath.Join(featuresPath, "devcontainer-features.json"))
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

func LoadDevContainerJson(applicationFolder string) DevContainerJson {
	// If we've already loaded devcontainer.json, load it
	devContainerJson, mapContainsKey := devcontainerJsonCache[applicationFolder]
	if mapContainsKey {
		return devContainerJson
	}

	// Load devcontainer.json
	if applicationFolder == "" {
		var err error
		applicationFolder, err = os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
	}
	content, dotDevContainerFolderErr := ioutil.ReadFile(filepath.Join(applicationFolder, "devcontainer", "devcontainer.json"))
	if dotDevContainerFolderErr != nil && os.IsNotExist(dotDevContainerFolderErr) {
		var dotDevContainerFileErr error
		content, dotDevContainerFileErr = ioutil.ReadFile(filepath.Join(applicationFolder, ".devcontainer.json"))
		if dotDevContainerFileErr != nil && os.IsNotExist(dotDevContainerFileErr) {
			log.Println("Folder", applicationFolder, "does not contain a .devcontainer folder / .devcontainer.json file")
			return devContainerJson
		}
	}
	err := json.Unmarshal(content, &devContainerJson)
	if err != nil {
		log.Fatal(err)
	}

	return devContainerJson
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

func GetOptionSelections(feature FeatureConfig, buildpackSettings BuildpackSettings, devContainerJson DevContainerJson) map[string]string {
	optionSelections := make(map[string]string)

	// Parse devcontainer.json features if file is found
	fullFeatureId := GetFullFeatureId(feature, buildpackSettings)
	for featureName, jsonOptionSelections := range devContainerJson.Features {
		if featureName == fullFeatureId || strings.HasPrefix(featureName, fullFeatureId+"@") {
			if reflect.TypeOf(jsonOptionSelections).String() == "string" {
				optionSelections["version"] = jsonOptionSelections.(string)
			} else {
				// Use reflection to convert the interface to a map[string]interface{} to a map[string]string
				mapRange := reflect.ValueOf(jsonOptionSelections).MapRange()
				for mapRange.Next() {
					optionSelections[mapRange.Key().String()] = mapRange.Value().Elem().String()
				}
			}
			break
		}
	}

	// Look for BP_CONTAINER_FEATURE_<feature.Id>_<option> environment variables, convert
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	for optionName := range feature.Options {
		optionNameSafe := strings.ReplaceAll(strings.ToUpper(optionName), "-", "_")
		optionValue := os.Getenv("BP_CONTAINER_FEATURE_" + idSafe + "_" + optionNameSafe)
		if optionValue != "" {
			optionSelections[optionName] = optionValue
		}
	}
	return optionSelections
}

func GetBuildEnvironment(feature FeatureConfig, optionSelections map[string]string, targetLayerPath string) []string {
	// Create environment that includes feature build args
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	optionEnvVarPrefix := "_BUILD_ARG_" + idSafe
	env := append(os.Environ(),
		optionEnvVarPrefix+"=true",
		"_FEATURE_BUILD_CONTEXT="+ContainerBuildContext())
	if targetLayerPath != "" {
		env = append(env, optionEnvVarPrefix+"_TARGET_PATH="+targetLayerPath)
	}
	for option, selection := range optionSelections {
		if selection != "" {
			env = append(env, optionEnvVarPrefix+"_"+strings.ToUpper(option)+"="+selection)
		}
	}
	return env
}

// e.g. chuxel/devcontainer/features/packcli
func GetFullFeatureId(feature FeatureConfig, buildpackSettings BuildpackSettings) string {
	return buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet + "/" + feature.Id
}
