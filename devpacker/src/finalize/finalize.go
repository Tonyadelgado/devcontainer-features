package finalize

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buildpacks/lifecycle/platform"
	"github.com/chuxel/devpacker-features/devpacker/common"
)

const labelMetadataTemplate = "{\"lifecycle\": {{ index .Config.Labels \"" + platform.LayerMetadataLabel + "\" }} , \"buildmode\": \"{{ index .Config.Labels \"" + common.BuildModeMetadataId + "\" }}\" , \"done\": \"{{ index .Config.Labels \"" + common.PostProcessingDoneMetadataId + "\" }}\" }"

type LabelBuldpackLayer struct {
	Data map[string]json.RawMessage
}

type PostProcessingMetadata struct {
	BuildMode          string                        `json:"buildmode"`
	PostProcessingDone string                        `json:"done"`
	Lifecycle          platform.LayersMetadataCompat `json:"lifecycle"`
}

//go:embed assets/post-processing.sh
var postProcessingScript []byte

//go:embed assets/post-processing.Dockerfile
var postProcessingDockerfile []byte

func FinalizeImage(imageToFinalize string, buildModeOverride string, applicationFolder string) {
	log.Println("Image to finalize:", imageToFinalize)
	log.Println("Application folder:", applicationFolder)

	// Get needed metadata from image label
	layerFeatureMetadataMap, buildMode, postProcessingDone := getImageFeatureMetadata(imageToFinalize, buildModeOverride)
	log.Println("Image build mode:", buildMode)

	// Execute post processing where required
	log.Println("Starting post processing. Already complete for: ", postProcessingDone)
	executePostProcessing(imageToFinalize, postProcessingDone, layerFeatureMetadataMap)

	// Create devcontainer.json and finalizer feature
	createDevContainerJson(imageToFinalize, layerFeatureMetadataMap, buildMode, applicationFolder)

	//log.Println("Calling devcontainer CLI to add remaining container features to image.")
	//devContainerImageBuild(imageToFinalize, targetDevContainerJsonPath, devContainerJsonPath)
}

func createDevContainerJson(imageToFinalize string, layerFeatureMetadataMap map[string]common.LayerFeatureMetadata, buildMode string, applicationFolder string) {
	// Get devcontainer.json as a map so we don't change any fields unexpectedly
	devContainerJsonMap := make(map[string]json.RawMessage)
	featureOptionSelections := make(map[string]interface{})
	var devContainerJsonPath string
	// Only load the existing devcontainer.json file if we're in the devcontainer context
	if buildMode == "devcontainer" {
		log.Println("Loading devcontainer.json if present.")
		devContainerJsonMap, devContainerJsonPath = common.LoadDevContainerJsonAsMap(applicationFolder)
		if devContainerJsonPath != "" {
			// Un-marshall devcontainer.json features into a map
			if err := json.Unmarshal(devContainerJsonMap["features"], &featureOptionSelections); err != nil {
				log.Fatal("Failed to unmarshal features from devcontainer.json: ", err)
			}
		}
	} else {
		log.Println("Skipping devcontainer.json load since build mode is", buildMode)
		devContainerJsonPath = common.FindDevContainerJson(applicationFolder)
	}

	// Remove any features from the in-bound devcontainer.json that we've already processed
	// remaining steps will be covered by the finalizer feature we'll generate next
	for featureId := range featureOptionSelections {
		if _, hasKey := layerFeatureMetadataMap[featureId]; hasKey || strings.HasPrefix(layerFeatureMetadataMap[featureId].Id, featureId+"@") {
			delete(featureOptionSelections, featureId)
		}
	}

	// Determine and create target folder paths
	targetFolder := applicationFolder
	if devContainerJsonPath == "" {
		targetFolder = filepath.Join(applicationFolder, ".devcontainer")
		devContainerJsonPath = filepath.Join(targetFolder, "devcontainer.json")
		if err := os.MkdirAll(targetFolder, 0755); err != nil {
			log.Fatal("Failed to create target folder: ", err)
		}
	} else {
		//targetFolder = filepath.Dir(devContainerJsonPath)
	}

	// Work around lack of local dev container feature reference support
	devContainerJsonMap = mergeFeatureConfigToDevContainerJson(devContainerJsonMap, layerFeatureMetadataMap)

	/* TODO: Use a feature instead of devcontainer.json merge once you can reference a local feature

	// Generate union of config for steps that have already been added to the image, output generated feature to target folder
	finalizeFeaturesJson := FeaturesJson{[]FeatureConfig{generateFinalizeFeatureConfig(layerFeatureMetadataMap)}}
	finalizeFeaturesJsonBytes, err := json.MarshalIndent(&finalizeFeaturesJson, "", "\t")
	if err != nil {
		log.Fatal("Failed to marshal finalizeFeaturesJson to json.RawMessage: ", err)
	}
	finalizeFeatureTargetFolder := filepath.Join(targetFolder, "devpack-config")
	if err := os.MkdirAll(finalizeFeatureTargetFolder, 0755); err != nil {
		log.Fatal("Failed to create finalizer feature target folder: ", err)
	}
	log.Println("Writing out updated related finalize feature to:", finalizeFeatureTargetFolder)
	WriteFile(filepath.Join(finalizeFeatureTargetFolder, "devcontainer-features.json"), finalizeFeaturesJsonBytes)
	WriteFile(filepath.Join(finalizeFeatureTargetFolder, "install.sh"), []byte("exit 0"))
	featureOptionSelections["./devpack-config#finalize"] = "latest"
	*/

	// Convert feature map back into a RawMessage, and add it back into the devcontainer json object
	featureRawMessage, err := json.Marshal(featureOptionSelections)
	if err != nil {
		log.Fatal("Failed to marshal devContainerJsonFeatureMap to json.RawMessage: ", err)
	}
	devContainerJsonMap["features"] = featureRawMessage
	devContainerJsonMap["image"] = common.ToJsonRawMessage(imageToFinalize)
	devContainerJsonMap["userEnvProbe"] = common.ToJsonRawMessage("loginInteractiveShell")
	delete(devContainerJsonMap, "build")
	delete(devContainerJsonMap, "dockerComposeFile")

	// Encode json content and write it to a file
	updatedDevContainerJsonBytes, err := json.MarshalIndent(devContainerJsonMap, "", "\t")
	if err != nil {
		log.Fatal("Failed to marshal devContainerJsonMap to json: ", err)
	}
	targetDevContainerJsonPath := devContainerJsonPath + ".devpack"
	log.Println("Writing out updated devcontainer.json file:", targetDevContainerJsonPath)
	if err := common.WriteFile(targetDevContainerJsonPath, updatedDevContainerJsonBytes); err != nil {
		log.Fatal("Failed to write updated devcontainer.json file: ", err)
	}
}

func generateFinalizeFeatureConfig(layerFeatureMetadataMap map[string]common.LayerFeatureMetadata) common.FeatureConfig {
	finalizeFeatureConfig := common.FeatureConfig{Id: "finalize"}
	// Merge in remaining config from features already in the image
	for _, layerFeatureMetadata := range layerFeatureMetadataMap {
		// Merge flags
		finalizeFeatureConfig.Privileged = finalizeFeatureConfig.Privileged || layerFeatureMetadata.Config.Privileged
		finalizeFeatureConfig.Init = finalizeFeatureConfig.Init || layerFeatureMetadata.Config.Init

		// Merge string arrays
		finalizeFeatureConfig.CapAdd = common.SliceUnion(finalizeFeatureConfig.CapAdd, layerFeatureMetadata.Config.CapAdd)
		finalizeFeatureConfig.SecurityOpt = common.SliceUnion(finalizeFeatureConfig.SecurityOpt, layerFeatureMetadata.Config.SecurityOpt)
		finalizeFeatureConfig.Extensions = common.SliceUnion(finalizeFeatureConfig.Extensions, layerFeatureMetadata.Config.Extensions)

		// Merge VS Code settings
		if finalizeFeatureConfig.Settings == nil {
			finalizeFeatureConfig.Settings = make(map[string]interface{})
		}
		for key, value := range layerFeatureMetadata.Config.Settings {
			finalizeFeatureConfig.Settings[key] = value
		}
		// Merge mount points
		for _, newMount := range layerFeatureMetadata.Config.Mounts {
			shouldAdd := true
			for _, mount := range finalizeFeatureConfig.Mounts {
				if mount.Source == newMount.Source && mount.Target == newMount.Target && mount.Type == newMount.Type {
					shouldAdd = false
					break
				}
			}
			if shouldAdd {
				layerFeatureMetadata.Config.Mounts = append(layerFeatureMetadata.Config.Mounts, newMount)
			}
		}
	}
	// Add required version option (not used)
	finalizeFeatureConfig.Options = map[string]common.FeatureOption{
		"version": {Type: "string", Enum: []string{"latest"}, Default: "latest", Description: "Not used."},
	}
	return finalizeFeatureConfig
}

func executePostProcessing(imageToFinalize string, postProcessingDone string, layerFeatureMetadataMap map[string]common.LayerFeatureMetadata) {
	var err error
	tempDir := filepath.Join(os.TempDir(), strconv.FormatInt(rand.Int63(), 36))
	if err = os.MkdirAll(tempDir, 0777); err != nil {
		log.Fatal("Failed to make temp directory: ", err)
	}

	postProcessingScriptPath := filepath.Join(tempDir, "post-processing.sh")
	if err = common.WriteFile(postProcessingScriptPath, postProcessingScript); err != nil {
		log.Fatal("Failed to write post-processing.sh: ", err)
	}

	// Append any needed post-processing steps to dockerfile
	postProcessingDockerfileModified := postProcessingDockerfile
	postProcessingRequired := ""
	postProcessingChecker := " " + postProcessingDone + " "
	for featureId, layerFeatureMetadata := range layerFeatureMetadataMap {
		postProcessingFeatureString := " " + featureId + " "
		if !strings.Contains(postProcessingChecker, postProcessingFeatureString) {
			postProcessingRequired += featureId + " "
			postProcessingDone += featureId + " "
			// Apply post processing for containerEnv
			for varName, varValue := range layerFeatureMetadata.Config.ContainerEnv {
				envVarSnippet := "\nENV " + varName + "=" + varValue
				postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte(envVarSnippet)...)
			}
		}
	}
	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err = common.WriteFile(dockerFilePath, postProcessingDockerfileModified); err != nil {
		log.Fatal("Failed to write Dockerfile: ", err)
	}

	dockerCli(tempDir, false, "build", "--no-cache", "--build-arg", "IMAGE_NAME="+imageToFinalize, "--build-arg", "POST_PROCESSING_REQUIRED="+postProcessingRequired, "--build-arg", "POST_PROCESSING_DONE="+strings.TrimSpace(postProcessingDone), "-t", imageToFinalize, "-f", dockerFilePath, ".")
	if err = os.RemoveAll(tempDir); err != nil {
		log.Fatal("Failed to remove temp directory: ", err)
	}
}

func dockerCli(workingDir string, captureOutput bool, args ...string) []byte {
	var outputBytes bytes.Buffer
	var errorOutput bytes.Buffer

	dockerCommand := exec.Command("docker", args...)
	dockerCommand.Env = os.Environ()
	if captureOutput {
		dockerCommand.Stdout = &outputBytes
		dockerCommand.Stderr = &errorOutput
	} else {
		writer := log.Writer()
		dockerCommand.Stdout = writer
		dockerCommand.Stderr = writer
	}
	if workingDir != "" {
		dockerCommand.Dir = workingDir
	}
	commandErr := dockerCommand.Run()
	if commandErr != nil || dockerCommand.ProcessState.ExitCode() != 0 || errorOutput.Len() != 0 {
		log.Fatal("Docker command failed: " + errorOutput.String() + commandErr.Error())
	}
	return outputBytes.Bytes()
}

// Inspect the specified image to get any feature config set on a label
func getImageFeatureMetadata(imageToFinalize string, buildModeOverride string) (map[string]common.LayerFeatureMetadata, string, string) {
	var labelMetadata PostProcessingMetadata

	// Use docker inspect to get metadata
	labelJsonBytes := dockerCli("", true, "image", "inspect", imageToFinalize, "-f", labelMetadataTemplate)

	// Parse contents
	if len(labelJsonBytes) > 0 && string(labelJsonBytes) != "" {
		if err := json.Unmarshal(labelJsonBytes, &labelMetadata); err != nil {
			log.Println("Unable to process feature metadata in image. Assuming no post processing is required.")
		}
	}

	// Set build mode
	buildMode := labelMetadata.BuildMode
	if buildModeOverride != "" {
		buildMode = buildModeOverride
	} else if labelMetadata.BuildMode == "" {
		// If no override, and we didn't get a value off of the image, then use the default
		buildMode = common.DefaultContainerImageBuildMode
	}

	// Convert feature metadata to map of LayerFeatureMetadata structs
	featureMetadataMap := make(map[string]common.LayerFeatureMetadata)
	if labelMetadata.Lifecycle.Buildpacks != nil {
		for _, buildpackMetadata := range labelMetadata.Lifecycle.Buildpacks {
			for _, buildpackLayerMetadata := range buildpackMetadata.Layers {
				if buildpackLayerMetadata.Data != nil {
					// Cast so we can use it
					data := buildpackLayerMetadata.Data.(map[string]interface{})
					if _, hasKey := data[common.FeatureLayerMetadataId]; hasKey {
						// Convert interface to struct
						featureMetadata := common.LayerFeatureMetadata{}
						featureMetadata.SetProperties(data[common.FeatureLayerMetadataId].(map[string]interface{}))
						featureMetadataMap[featureMetadata.Id] = featureMetadata
					}

				}
			}
		}
	}

	return featureMetadataMap, buildMode, labelMetadata.PostProcessingDone
}

func devContainerImageBuild(imageToFinalize string, tempDevContainerJsonPath string, originalDevContainerJsonPath string) {
	workingDir := filepath.Dir(tempDevContainerJsonPath)

	// Rename files so temp devcontainer.json is used
	if err := os.Rename(originalDevContainerJsonPath, originalDevContainerJsonPath+".orig"); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(tempDevContainerJsonPath, originalDevContainerJsonPath); err != nil {
		log.Fatal(err)
	}

	// Invoke dev container CLI
	dockerCommand := exec.Command("devcontainer", "build", "--image-name", imageToFinalize)
	dockerCommand.Env = os.Environ()
	writer := log.Writer()
	dockerCommand.Stdout = writer
	dockerCommand.Stderr = writer
	dockerCommand.Dir = workingDir
	commandErr := dockerCommand.Run()

	// Rename files back
	if err := os.Rename(originalDevContainerJsonPath, tempDevContainerJsonPath); err != nil {
		log.Fatal(err)
	}
	if err := os.Rename(originalDevContainerJsonPath+".orig", originalDevContainerJsonPath); err != nil {
		log.Fatal(err)
	}

	// Report command error if there was one
	if commandErr != nil || dockerCommand.ProcessState.ExitCode() != 0 {
		log.Fatal("Failed to build using devcontainer CLI: " + commandErr.Error())
	}
}
