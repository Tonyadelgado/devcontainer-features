package finalize

import (
	_ "embed"
	"encoding/json"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buildpacks/lifecycle/platform"
	"github.com/chuxel/devpacker-features/devpacker/common"
)

const labelMetadataTemplate = "{\"layers\": {{ index .Config.Labels \"" + platform.LayerMetadataLabel + "\" }} , \"buildmode\": \"{{ index .Config.Labels \"" + common.BuildModeMetadataId + "\" }}\" , \"done\": \"{{ index .Config.Labels \"" + common.PostProcessingDoneMetadataId + "\" }}\" , \"entrypoint\": {{ json .Config.Entrypoint }} }"

type LabelBuldpackLayer struct {
	Data map[string]json.RawMessage
}

type DockerImageMetadata struct {
	BuildMode     string                        `json:"buildmode"`
	AlreadyDone   string                        `json:"done"`
	LayerMetadata platform.LayersMetadataCompat `json:"layers"`
	Entrypoint    []string                      `json:"entrypoint"`
}

type PostProcessingConfig struct {
	Image                string
	ApplicationFolder    string
	BuildMode            string
	AlreadyDone          string
	LayerFeatureMetadata map[string]common.LayerFeatureMetadata
	Entrypoint           []string
}

//go:embed assets/post-processing.sh
var postProcessingScript []byte

//go:embed assets/post-processing.Dockerfile
var postProcessingDockerfile []byte

func FinalizeImage(imageToFinalize string, buildModeOverride string, applicationFolder string) {
	log.Println("Image to finalize:", imageToFinalize)
	log.Println("Application folder:", applicationFolder)

	// Get needed metadata from image label
	postProcessingConfig := newPostProcessingConfig(imageToFinalize, buildModeOverride, applicationFolder)
	log.Println("Image build mode:", postProcessingConfig.BuildMode)

	// Execute post processing where required
	log.Println("Starting post processing. Already complete for: ", postProcessingConfig.AlreadyDone)
	executePostProcessing(postProcessingConfig)

	// Create devcontainer.json and finalizer feature
	createDevContainerJson(postProcessingConfig)

	/* TODO: Chain call to devcontainer CLI when it works without pulling an image
	log.Println("Calling devcontainer CLI to add remaining container features to image.")
	devContainerImageBuild(imageToFinalize, targetDevContainerJsonPath, devContainerJsonPath)
	*/
}

func createDevContainerJson(postProcessingConfig PostProcessingConfig) {
	// Get devcontainer.json as a map so we don't change any fields unexpectedly
	devContainerJsonMap := make(map[string]json.RawMessage)
	featureOptionSelections := make(map[string]interface{})
	var devContainerJsonPath string
	// Only load the existing devcontainer.json file if we're in the devcontainer context
	if postProcessingConfig.BuildMode == "devcontainer" {
		log.Println("Loading devcontainer.json if present.")
		devContainerJsonMap, devContainerJsonPath = common.LoadDevContainerJsonAsMap(postProcessingConfig.ApplicationFolder)
		if devContainerJsonPath != "" {
			// Un-marshall devcontainer.json features into a map
			if err := json.Unmarshal(devContainerJsonMap["features"], &featureOptionSelections); err != nil {
				log.Fatal("Failed to unmarshal features from devcontainer.json: ", err)
			}
		}
	} else {
		log.Println("Skipping devcontainer.json load since build mode is", postProcessingConfig.BuildMode)
		devContainerJsonPath = common.FindDevContainerJson(postProcessingConfig.ApplicationFolder)
	}

	// Remove any features from the in-bound devcontainer.json that we've already processed
	// remaining steps will be covered by the finalizer feature we'll generate next
	for featureId := range featureOptionSelections {
		if _, hasKey := postProcessingConfig.LayerFeatureMetadata[featureId]; hasKey || strings.HasPrefix(postProcessingConfig.LayerFeatureMetadata[featureId].Id, featureId+"@") {
			delete(featureOptionSelections, featureId)
		}
	}

	// Determine and create target folder paths
	targetFolder := postProcessingConfig.ApplicationFolder
	if devContainerJsonPath == "" {
		targetFolder = filepath.Join(postProcessingConfig.ApplicationFolder, ".devcontainer")
		devContainerJsonPath = filepath.Join(targetFolder, "devcontainer.json")
		if err := os.MkdirAll(targetFolder, 0755); err != nil {
			log.Fatal("Failed to create target folder: ", err)
		}
	} else {
		//targetFolder = filepath.Dir(devContainerJsonPath)
	}

	// TODO: Use a feature instead of devcontainer.json merge once you can reference a local feature
	devContainerJsonMap = mergeFeatureConfigToDevContainerJson(postProcessingConfig, devContainerJsonMap)

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
	devContainerJsonMap["image"] = common.ToJsonRawMessage(postProcessingConfig.Image)
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

func generateFinalizeFeatureConfig(postProcessingConfig PostProcessingConfig) common.FeatureConfig {
	finalizeFeatureConfig := common.FeatureConfig{Id: "finalize"}
	// Merge in remaining config from features already in the image
	for _, layerFeatureMetadata := range postProcessingConfig.LayerFeatureMetadata {
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

func executePostProcessing(postProcessingConfig PostProcessingConfig) {
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
	postProcessingChecker := " " + postProcessingConfig.AlreadyDone + " "
	for featureId, layerFeatureMetadata := range postProcessingConfig.LayerFeatureMetadata {
		postProcessingFeatureString := " " + featureId + " "
		if !strings.Contains(postProcessingChecker, postProcessingFeatureString) {
			postProcessingRequired += featureId + " "
			postProcessingConfig.AlreadyDone += featureId + " "
			// Apply post processing for containerEnv
			for varName, varValue := range layerFeatureMetadata.Config.ContainerEnv {
				envVarSnippet := "\nENV " + varName + "=" + varValue
				postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte(envVarSnippet)...)
			}
		}
	}
	postProcessingConfig.AlreadyDone = strings.TrimSpace(postProcessingConfig.AlreadyDone)

	// Append update entrypoint in Dockerfile if required
	if !common.SliceContainsString(postProcessingConfig.Entrypoint, common.CommonEntrypointDBootstrapPath) {
		postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte("\n\nENTRYPOINT [\""+common.CommonEntrypointDBootstrapPath+"\"")...)
		for _, argument := range postProcessingConfig.Entrypoint {
			argument = strings.ReplaceAll(argument, "\"", "\\\"")
			postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte(",\""+argument+"\"")...)
		}
		postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte("]")...)
	}

	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err = common.WriteFile(dockerFilePath, postProcessingDockerfileModified); err != nil {
		log.Fatal("Failed to write Dockerfile: ", err)
	}

	common.DockerCli(tempDir, false, "build",
		"--no-cache",
		"--build-arg", "IMAGE_NAME="+postProcessingConfig.Image,
		"--build-arg", "POST_PROCESSING_REQUIRED="+postProcessingRequired,
		"--build-arg", "POST_PROCESSING_DONE="+postProcessingConfig.AlreadyDone,
		"-t", postProcessingConfig.Image,
		"-f", dockerFilePath, ".")

	if err = os.RemoveAll(tempDir); err != nil {
		log.Fatal("Failed to remove temp directory: ", err)
	}
}

// Inspect the specified image and create a new instance of PostProcessingConfig
func newPostProcessingConfig(imageToFinalize string, buildModeOverride string, applicationFolder string) PostProcessingConfig {
	// Use docker inspect to get metadata
	var dockerImageMetadata DockerImageMetadata
	inspectJsonBytes := common.DockerCli("", true, "image", "inspect", imageToFinalize, "-f", labelMetadataTemplate)
	if len(inspectJsonBytes) > 0 && string(inspectJsonBytes) != "" {
		if err := json.Unmarshal(inspectJsonBytes, &dockerImageMetadata); err != nil {
			log.Println("Unable to process feature metadata in image. Assuming no post processing is required.")
		}
	}

	postProcessingConfig := PostProcessingConfig{
		Image:             imageToFinalize,
		BuildMode:         dockerImageMetadata.BuildMode,
		AlreadyDone:       dockerImageMetadata.AlreadyDone,
		ApplicationFolder: applicationFolder,
		Entrypoint:        dockerImageMetadata.Entrypoint,
	}

	// Set build mode
	if buildModeOverride != "" {
		postProcessingConfig.BuildMode = buildModeOverride
	} else if dockerImageMetadata.BuildMode == "" {
		// If no override, and we didn't get a value off of the image, then use the default
		postProcessingConfig.BuildMode = common.DefaultContainerImageBuildMode
	}

	// Convert feature metadata to map of LayerFeatureMetadata structs
	postProcessingConfig.LayerFeatureMetadata = make(map[string]common.LayerFeatureMetadata)
	if dockerImageMetadata.LayerMetadata.Buildpacks != nil {
		for _, buildpackMetadata := range dockerImageMetadata.LayerMetadata.Buildpacks {
			for _, buildpackLayerMetadata := range buildpackMetadata.Layers {
				if buildpackLayerMetadata.Data != nil {
					// Cast so we can use it
					data := buildpackLayerMetadata.Data.(map[string]interface{})
					if _, hasKey := data[common.FeatureLayerMetadataId]; hasKey {
						// Convert interface to struct
						featureMetadata := common.LayerFeatureMetadata{}
						featureMetadata.SetProperties(data[common.FeatureLayerMetadataId].(map[string]interface{}))
						postProcessingConfig.LayerFeatureMetadata[featureMetadata.Id] = featureMetadata
					}

				}
			}
		}
	}

	return postProcessingConfig
}

/* TODO: Uncomment once you can use the devcontainer CI without pulling an image

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
*/
