package main

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
)

const labelMetadataTemplate = "{\"lifecycle\": {{ index .Config.Labels \"io.buildpacks.lifecycle.metadata\" }} , \"buildmode\": \"{{ index .Config.Labels \"" + BuildModeMetadataId + "\" }}\" , \"done\": \"{{ index .Config.Labels \"" + PostProcessingDoneMetadataId + "\" }}\" }"

type LabelBuldpack struct {
	Layers map[string]LabelBuldpackLayer
}

type LabelBuldpackLayer struct {
	Data map[string]json.RawMessage
}

type LifecycleLabelMetadata struct {
	Buildpacks []LabelBuldpack
}

type LabelMetadata struct {
	BuildMode          string                 `json:"buildmode"`
	PostProcessingDone string                 `json:"done"`
	Lifecycle          LifecycleLabelMetadata `json:"lifecycle"`
}

//go:embed assets/post-processing.sh
var postProcessingScript []byte

//go:embed assets/post-processing.Dockerfile
var postProcessingDockerfile []byte

func FinalizeImage(imageToFinalize string, buildMode string, applicationFolder string) {
	log.Println("Image to finalize:", imageToFinalize)
	log.Println("Application folder:", applicationFolder)

	// Get needed metadata from image label
	labelMetadata := getImageLabelMetadata(imageToFinalize, buildMode)
	log.Println("Image build mode:", labelMetadata.BuildMode)

	// Get devcontainer.json as a map so we don't change any fields unexpectedly
	devContainerJsonMap := make(map[string]json.RawMessage)
	devContainerJsonFeatureMap := make(map[string]interface{})
	var devContainerJsonPath string
	// Only load the existing devcontainer.json file if we're in the devcontainer context
	if labelMetadata.BuildMode == "devcontainer" {
		log.Println("Loading devcontainer.json if present.")
		devContainerJsonMap, devContainerJsonPath = LoadDevContainerJsonAsMap(applicationFolder)
		// Un-marshall devcontainer.json features into a map
		if err := json.Unmarshal(devContainerJsonMap["features"], &devContainerJsonFeatureMap); err != nil {
			log.Fatal("Failed to unmarshal features from devcontainer.json: ", err)
		}
	} else {
		log.Println("Skipping devcontainer.json load since build mode is", labelMetadata.BuildMode)
		devContainerJsonPath = FindDevContainerJson(applicationFolder)
	}

	// Inspect the specified image to and add any feature metadata into our features map
	log.Println("Inspecting image lifecycle metadata", imageToFinalize, "for feature metadata.")
	var layerFeatureMetadataList []LayerFeatureMetadata
	layerFeatureMetadataList, devContainerJsonFeatureMap = convertLifecycleMetadataToFeatureOptionSelections(labelMetadata, devContainerJsonFeatureMap)

	// Execute post processing where required
	log.Println("Starting post processing. Already complete for: ", labelMetadata.PostProcessingDone)
	executePostProcessing(imageToFinalize, labelMetadata.PostProcessingDone, layerFeatureMetadataList)

	// Convert feature map back into a RawMessage, and add it back into the devcontainer json object
	featureRawMessage, err := json.Marshal(devContainerJsonFeatureMap)
	if err != nil {
		log.Fatal("Failed to marshal devContainerJsonFeatureMap to json.RawMessage: ", err)
	}
	devContainerJsonMap["features"] = featureRawMessage

	// If no devcontainer.json exists, create one with the image property set to the specified image
	targetDevContainerJsonPath := devContainerJsonPath
	if targetDevContainerJsonPath == "" {
		targetDevContainerJsonPath = filepath.Join(applicationFolder, ".devcontainer.json")
		devContainerJsonMap["image"] = toJsonRawMessage(imageToFinalize)
	}

	// Always force userEnvProbe to interactiveLoginShell
	devContainerJsonMap["userEnvProbe"] = toJsonRawMessage("loginInteractiveShell")

	// Append ".buildpack" to avoid overwriting
	targetDevContainerJsonPath += ".buildpack"

	// Encode json content and write it to a file
	updatedDevContainerJsonContent, err := json.MarshalIndent(devContainerJsonMap, "", "\t")
	if err != nil {
		log.Fatal("Failed to marshal devContainerJsonMap to json: ", err)
	}
	log.Println("Writing out updated devcontainer.json file:", targetDevContainerJsonPath)
	if err := WriteFile(targetDevContainerJsonPath, updatedDevContainerJsonContent); err != nil {
		log.Fatal("Failed to write updated devcontainer.json file: ", err)
	}

	//log.Println("Calling devcontainer CLI to add remaining container features to image.")
	//devContainerImageBuild(imageToFinalize, targetDevContainerJsonPath, devContainerJsonPath)

}

func executePostProcessing(imageToFinalize string, postProcessingDone string, layerFeatureMetadataList []LayerFeatureMetadata) {
	var err error
	tempDir := filepath.Join(os.TempDir(), strconv.FormatInt(rand.Int63(), 36))
	if err = os.MkdirAll(tempDir, 0777); err != nil {
		log.Fatal("Failed to make temp directory: ", err)
	}

	postProcessingScriptPath := filepath.Join(tempDir, "post-processing.sh")
	if err = WriteFile(postProcessingScriptPath, postProcessingScript); err != nil {
		log.Fatal("Failed to write post-processing.sh: ", err)
	}

	// Append any needed post-processing steps to dockerfile
	postProcessingDockerfileModified := postProcessingDockerfile
	postProcessingRequired := ""
	postProcessingChecker := " " + postProcessingDone + " "
	for _, layerFeatureMetadata := range layerFeatureMetadataList {
		postProcessingFeatureString := " " + layerFeatureMetadata.Id + " "
		if !strings.Contains(postProcessingChecker, postProcessingFeatureString) {
			postProcessingRequired += layerFeatureMetadata.Id + " "
			postProcessingDone += layerFeatureMetadata.Id + " "
			// Apply post processing for containerEnv
			for varName, varValue := range layerFeatureMetadata.Config.ContainerEnv {
				envVarSnippet := "\nENV " + varName + "=" + varValue
				postProcessingDockerfileModified = append(postProcessingDockerfileModified, []byte(envVarSnippet)...)
			}
		}
	}
	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err = WriteFile(dockerFilePath, postProcessingDockerfileModified); err != nil {
		log.Fatal("Failed to write Dockerfile: ", err)
	}

	dockerCli(tempDir, false, "build", "--no-cache", "--build-arg", "IMAGE_NAME="+imageToFinalize, "--build-arg", "POST_PROCESSING_REQUIRED="+postProcessingRequired, "--build-arg", "POST_PROCESSING_DONE="+strings.TrimSpace(postProcessingDone), "-t", imageToFinalize, "-f", dockerFilePath, ".")
	if err = os.RemoveAll(tempDir); err != nil {
		log.Fatal("Failed to remove temp directory: ", err)
	}
}

func convertLifecycleMetadataToFeatureOptionSelections(labelMetadata LabelMetadata, devContainerJsonFeatureMap map[string]interface{}) ([]LayerFeatureMetadata, map[string]interface{}) {
	layerFeatureMetadataList := []LayerFeatureMetadata{}
	// For each buildpack
	for _, buildpackMetadata := range labelMetadata.Lifecycle.Buildpacks {
		// And each layer in each buildpack
		for _, layerMetadata := range buildpackMetadata.Layers {
			// See if there is any feature metadata
			if layerFeatureMetadataRaw, hasKey := layerMetadata.Data[FeatureLayerMetadataId]; hasKey {
				// If so, load the json contents
				var layerFeatureMetadata LayerFeatureMetadata
				if err := json.Unmarshal(layerFeatureMetadataRaw, &layerFeatureMetadata); err != nil {
					log.Fatal("Failed to unmarshal dev container feature metadata: ", err)
				}
				// Add the feature metadata to the list
				layerFeatureMetadataList = append(layerFeatureMetadataList, layerFeatureMetadata)
				// And compare remove the related feature from the passed in feature selections if present
				for featureId := range devContainerJsonFeatureMap {
					if featureId == layerFeatureMetadata.Id || strings.HasPrefix(featureId, layerFeatureMetadata.Id+"@") {
						delete(devContainerJsonFeatureMap, featureId)
						break
					}
				}
				// And finally add the selections
				devContainerJsonFeatureMap[layerFeatureMetadata.Id+"@"+layerFeatureMetadata.Version] = layerFeatureMetadata.OptionSelections
			}
		}
	}
	return layerFeatureMetadataList, devContainerJsonFeatureMap
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

func toJsonRawMessage(value interface{}) json.RawMessage {
	var err error
	var bytes json.RawMessage
	if bytes, err = json.Marshal(value); err != nil {
		log.Fatal("Failed to convert to json.RawMessage:", err)
	}
	return bytes
}

// Inspect the specified image to get any feature config set on a label
func getImageLabelMetadata(imageToFinalize string, buildModeOverride string) LabelMetadata {
	var labelMetadata LabelMetadata

	// Use docker inspect to get metadata
	labelJsonBytes := dockerCli("", true, "image", "inspect", imageToFinalize, "-f", labelMetadataTemplate)

	// Parse contents
	if len(labelJsonBytes) > 0 && string(labelJsonBytes) != "" {
		if err := json.Unmarshal(labelJsonBytes, &labelMetadata); err != nil {
			log.Println("Unable to process feature metadata in image. Assuming no post processing is required.")
		}
	}

	// Set build mode
	if buildModeOverride != "" {
		labelMetadata.BuildMode = buildModeOverride
	} else if labelMetadata.BuildMode == "" {
		// If no override, and we didn't get a value off of the image, then use the default
		labelMetadata.BuildMode = DefaultContainerImageBuildMode
	}

	return labelMetadata
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

func generateDevContainerFeatureWrapper(devContainerJsonFeatureMap map[string]interface{}) {

}
