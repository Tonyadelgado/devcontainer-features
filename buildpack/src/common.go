package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

const DefaultApiVersion = "0.7"
const MetadataIdPrefix = "com.microsoft.devcontainer"
const FeaturesetMetadataId = "featureset"
const FeaturesMetadataId = "features"
const AppliedFeaturesLabelId = MetadataIdPrefix + ".features"

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

func loadDevContainerJsonConent(applicationFolder string) ([]byte, string) {
	devContainerJsonPath := FindDevContainerJson(applicationFolder)
	if devContainerJsonPath == "" {
		return []byte{}, devContainerJsonPath
	}

	content, err := ioutil.ReadFile(devContainerJsonPath)
	if err != nil {
		log.Fatal(err)
	}
	return content, devContainerJsonPath
}

func LoadDevContainerJson(applicationFolder string) (DevContainerJson, string) {
	var devContainerJson DevContainerJson
	content, devContainerJsonPath := loadDevContainerJsonConent(applicationFolder)
	err := json.Unmarshal(content, &devContainerJson)
	if err != nil {
		log.Fatal(err)
	}
	return devContainerJson, devContainerJsonPath
}

func LoadDevContainerJsonAsMap(applicationFolder string) (map[string]json.RawMessage, string) {
	var jsonMap map[string]json.RawMessage
	content, devContainerJsonPath := loadDevContainerJsonConent(applicationFolder)
	err := json.Unmarshal(content, &jsonMap)
	if err != nil {
		log.Fatal(err)
	}
	return jsonMap, devContainerJsonPath
}

func GetFeatureScriptPath(buidpackPath string, featureId string, script string) string {
	return filepath.Join(buidpackPath, "features", featureId, "bin", script)
}

func GetContainerImageBuildContext() string {
	context := os.Getenv("BP_CONTAINER_FEATURE_BUILD_CONTEXT")
	if context == "" {
		return "devcontainer"
	}
	return context
}

func GetOptionSelections(feature FeatureConfig, buildpackSettings BuildpackSettings, devContainerJson DevContainerJson) map[string]string {
	optionSelections := make(map[string]string)

	// If in dev container mode, parse devcontainer.json features (if any)
	if GetContainerImageBuildContext() == "devcontainer" {
		fullFeatureId := GetFullFeatureId(feature, buildpackSettings)
		for featureName, jsonOptionSelections := range devContainerJson.Features {
			if featureName == fullFeatureId || strings.HasPrefix(featureName, fullFeatureId+"@") {
				if reflect.TypeOf(jsonOptionSelections).String() == "string" {
					optionSelections["version"] = jsonOptionSelections.(string)
				} else {
					// Use reflection to convert the from a map[string]interface{} to a map[string]string
					mapRange := reflect.ValueOf(jsonOptionSelections).MapRange()
					for mapRange.Next() {
						optionSelections[mapRange.Key().String()] = mapRange.Value().Elem().String()
					}
				}
				break
			}
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
		"_FEATURE_BUILD_CONTEXT="+GetContainerImageBuildContext())
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

func CpR(sourcePath string, targetFolderPath string) {
	sourceFileInfo, err := os.Stat(sourcePath)
	if err != nil {
		// Return if source path doesn't exist so we can use this with optional files
		return
	}
	// Handle if source is file
	if !sourceFileInfo.IsDir() {
		Cp(sourcePath, targetFolderPath)
		return
	}

	// Otherwise create the directory and scan contents
	toFolderPath := filepath.Join(targetFolderPath, sourceFileInfo.Name())
	os.MkdirAll(toFolderPath, sourceFileInfo.Mode())
	fileInfos, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(sourcePath, fileInfo.Name())
		if fileInfo.IsDir() {
			CpR(fromPath, toFolderPath)
		} else {
			Cp(fromPath, toFolderPath)
		}
	}
}

func Cp(sourceFilePath string, targetFolderPath string) {
	sourceFileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Make target folder
	targetFilePath := filepath.Join(targetFolderPath, sourceFileInfo.Name())
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		log.Fatal(err)
	}
	// Sync source and target file mode and ownership
	targetFile.Chmod(sourceFileInfo.Mode())
	targetFile.Chown(int(sourceFileInfo.Sys().(*syscall.Stat_t).Uid), int(sourceFileInfo.Sys().(*syscall.Stat_t).Gid))

	// Execute copy
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		log.Fatal(err)
	}
	targetFile.Close()
	sourceFile.Close()
}

func WriteFile(filename string, fileBytes []byte) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err = file.Write(fileBytes); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return nil
}
