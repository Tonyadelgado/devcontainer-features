package common

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

	"github.com/tailscale/hujson"
	"gonum.org/v1/gonum/stat/combin"
)

var cachedContainerImageBuildMode = ""

type NonZeroExitError struct {
	ExitCode int
}

func (err NonZeroExitError) Error() string {
	return "Non-zero exit code: " + strconv.FormatInt(int64(err.ExitCode), 10)
}

type FeatureMount struct {
	Source string `json:"source,omitempty"`
	Target string `json:"target,omitempty"`
	Type   string `json:"type,omitempty"`
}

type FeatureOption struct {
	Type        string      `json:"type,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Proposals   []string    `json:"proposals,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description"`
}

type FeatureConfig struct {
	Id           string                   `json:"id,omitempty"`
	Name         string                   `json:"name,omitempty"`
	Options      map[string]FeatureOption `json:"options,omitempty"`
	Extensions   []string                 `json:"extensions,omitempty"`
	Settings     map[string]interface{}   `json:"settings,omitempty"`
	Entrypoint   string                   `json:"entrypoint,omitempty"`
	Privileged   bool                     `json:"privileged,omitempty"`
	Init         bool                     `json:"init,omitempty"`
	ContainerEnv map[string]string        `json:"containerEnv,omitempty"`
	Mounts       []FeatureMount           `json:"mounts,omitempty"`
	CapAdd       []string                 `json:"capAdd,omitempty"`
	SecurityOpt  []string                 `json:"securityOpt,omitempty"`
	BuildArg     string                   `json:"buildArg,omitempty"`
}

func (fc *FeatureConfig) SetProperties(propertyMap map[string]interface{}) {
	for property, value := range propertyMap {
		if value != nil {
			switch property {
			case "Mounts":
				out := []FeatureMount{}
				inputInterfaceArray := value.([]map[string]interface{})
				for _, value := range inputInterfaceArray {
					obj := propertyMapToInterface(value, reflect.TypeOf(FeatureMount{})).(*FeatureMount)
					out = append(out, *obj)
				}
				fc.Mounts = out
			case "Options":
				// Convert map[string]interface{} to map[string]FeatureOption
				out := make(map[string]FeatureOption)
				inputInterfaceMap := value.(map[string]interface{})
				for key, value := range inputInterfaceMap {
					valuePropertyMap := value.(map[string]interface{})
					obj := propertyMapToInterface(valuePropertyMap, reflect.TypeOf(FeatureOption{})).(*FeatureOption)
					out[key] = *obj
				}
				fc.Options = out
			default:
				field := reflect.ValueOf(fc).Elem().FieldByName(property)
				setFieldValue(field, value)
			}
		}
	}
}

type FeaturesJson struct {
	Features []FeatureConfig `json:"features"`
}

// Required configuration for processing
type DevpackSettings struct {
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

type LayerFeatureMetadata struct {
	Id               string
	Version          string
	Config           FeatureConfig
	OptionSelections map[string]string
}

func (lfm *LayerFeatureMetadata) SetProperties(propertyMap map[string]interface{}) {
	for property, value := range propertyMap {
		if value != nil {
			if property == "Config" {
				out := FeatureConfig{}
				inputInterfaceMap := value.(map[string]interface{})
				out.SetProperties(inputInterfaceMap)
				lfm.Config = out
			} else {
				field := reflect.ValueOf(lfm).Elem().FieldByName(property)
				setFieldValue(field, value)
			}
		}
	}
}

func propertyMapToInterface(propertyMap map[string]interface{}, typ reflect.Type) interface{} {
	objValue := reflect.New(typ)
	objValueElem := objValue.Elem()
	for key, value := range propertyMap {
		field := objValueElem.FieldByName(key)
		setFieldValue(field, value)
	}
	return objValue.Interface()
}

func setFieldValue(field reflect.Value, value interface{}) {
	reflectValue := reflect.ValueOf(value)
	reflectValueType := reflectValue.Type().Kind().String()
	switch reflectValueType {
	case "slice":
		convertedSliceValue := reflect.MakeSlice(field.Type(), 0, reflectValue.Len())
		for _, sliceItem := range value.([]interface{}) {
			convertedSliceValue = reflect.Append(convertedSliceValue, reflect.ValueOf(sliceItem))
		}
		field.Set(convertedSliceValue)
	case "map":
		convertedMapValue := reflect.MakeMap(field.Type())
		for key, value := range value.(map[string]interface{}) {
			convertedMapValue.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))
		}
		field.Set(convertedMapValue)
	default:
		field.Set(reflectValue.Convert(field.Type()))
	}
}

func LoadFeaturesJson(featuresPath string) FeaturesJson {
	// Load devcontainer-features.json or features.json
	if featuresPath == "" {
		featuresPath = os.Getenv(BuildpackDirEnvVar)
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

func LoadDevpackSettings(featuresPath string) DevpackSettings {
	if featuresPath == "" {
		featuresPath = os.Getenv(BuildpackDirEnvVar)
	}
	content, err := ioutil.ReadFile(filepath.Join(featuresPath, DevpackSettingsFilename))
	if err != nil {
		log.Fatal(err)
	}
	var jsonContents DevpackSettings
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
	// Strip out comments to enable parsing
	ast, err := hujson.Parse(content)
	if err != nil {
		log.Fatal(err)
	}
	ast.Standardize()
	content = ast.Pack()

	return content, devContainerJsonPath
}

func LoadDevContainerJson(applicationFolder string) (DevContainerJson, string) {
	var devContainerJson DevContainerJson
	content, devContainerJsonPath := loadDevContainerJsonConent(applicationFolder)
	if devContainerJsonPath != "" {
		err := json.Unmarshal(content, &devContainerJson)
		if err != nil {
			log.Fatal(err)
		}

	}
	return devContainerJson, devContainerJsonPath
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

func GetFeatureScriptPath(buidpackPath string, featureId string, script string) string {
	return filepath.Join(buidpackPath, "features", featureId, "bin", script)
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

func GetBuildEnvironment(feature FeatureConfig, optionSelections map[string]string, additionalVariables map[string]string) []string {
	// Create environment that includes feature build args
	env := append(os.Environ(),
		GetOptionEnvVarName(OptionSelectionEnvVarPrefix, feature.Id, "")+"=true")
	for optionId, selection := range optionSelections {
		if selection != "" {
			env = append(env, GetOptionEnvVarName(OptionSelectionEnvVarPrefix, feature.Id, optionId)+"="+selection)
		}
	}
	for varName, varValue := range additionalVariables {
		env = append(env, GetOptionEnvVarName(OptionSelectionEnvVarPrefix, feature.Id, varName)+"="+varValue)
	}
	log.Println(env)
	return env
}

func GetOptionEnvVarName(prefix string, featureId string, optionId string) string {
	if prefix == "" {
		prefix = OptionSelectionEnvVarPrefix
	}
	featureIdSafe := strings.ReplaceAll(strings.ToUpper(featureId), "-", "_")
	name := prefix + featureIdSafe
	if optionId != "" {
		optionIdSafe := strings.ReplaceAll(strings.ToUpper(optionId), "-", "_")
		name = prefix + featureIdSafe + "_" + strings.ToUpper(strings.ReplaceAll(optionIdSafe, "-", "_"))
	}
	return name
}

func GetOptionMetadataKey(optionId string) string {
	return OptionMetadataKeyPrefix + strings.ToLower(strings.ReplaceAll(optionId, "-", "_"))
}

// e.g. chuxel/devcontainer/features/packcli
func GetFullFeatureId(feature FeatureConfig, devpackSettings DevpackSettings, separator string) string {
	if separator == "" {
		separator = "/"
	}
	return devpackSettings.Publisher + separator + devpackSettings.FeatureSet + separator + feature.Id
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

	// Make target file
	targetFilePath := filepath.Join(targetFolderPath, sourceFileInfo.Name())
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		log.Fatal(err)
	}
	// Sync source and target file mode and ownership
	targetFile.Chmod(sourceFileInfo.Mode())
	SyncUIDGID(targetFile, sourceFileInfo)

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

func GetAllCombinations(arraySize int) [][]int {
	combinationList := [][]int{}
	for i := 1; i <= arraySize; i++ {
		combinationList = append(combinationList, combin.Combinations(arraySize, i)...)
	}
	return combinationList
}

func AddToSliceIfUnique(slice []string, value string) []string {
	if SliceContainsString(slice, value) {
		return slice
	}
	return append(slice, value)
}

func SliceContainsString(slice []string, item string) bool {
	for _, sliceItem := range slice {
		if sliceItem == item {
			return true
		}
	}
	return false
}

func SliceUnion(slice1 []string, slice2 []string) []string {
	union := slice1[0:]
	for _, sliceItem := range slice2 {
		union = AddToSliceIfUnique(union, sliceItem)
	}
	return union
}

func ToJsonRawMessage(value interface{}) json.RawMessage {
	var err error
	var bytes json.RawMessage
	if bytes, err = json.Marshal(value); err != nil {
		log.Fatal("Failed to convert to json.RawMessage:", err)
	}
	return bytes
}
