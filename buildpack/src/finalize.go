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

type LabelBuldpack struct {
	Layers map[string]LabelBuldpackLayer
}

type LabelBuldpackLayer struct {
	Data map[string]json.RawMessage
}

type LifecycleLabelMetadata struct {
	Buildpacks []LabelBuldpack
}

//go:embed assets/ensure-launcher-env.sh
var ensureLauncherEnvScript []byte

//go:embed assets/ensure-launcher-env.Dockerfile
var envRestoreDockerfile []byte

func FinalizeImage(imageToFinalize string, applicationFolder string) {
	buildMode := GetContainerImageBuildMode()
	log.Println("Image to finalize:", imageToFinalize)
	log.Println("Image build mode:", buildMode)
	log.Println("Application folder:", applicationFolder)

	// Get devcontainer.json as a map so we don't change any fields unexpectedly
	var devContainerJsonMap map[string]json.RawMessage
	var devContainerJsonFeatureMap map[string]interface{}
	var devContainerJsonPath string
	// Only load the existing devcontainer.json file if we're in the devcontainer context
	if buildMode == "devcontainer" {
		log.Println("Loading devcontainer.json if present.")
		devContainerJsonMap, devContainerJsonPath = LoadDevContainerJsonAsMap(applicationFolder)
		// Unmarshall devcontainer.json features into a map
		if err := json.Unmarshal(devContainerJsonMap["features"], &devContainerJsonFeatureMap); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Skipping devcontainer.json load since build mode is", buildMode)
		devContainerJsonPath = FindDevContainerJson(applicationFolder)
	}

	// Inspect the specified image to and add any feature metadata into our features map
	log.Println("Inspecting image lifecycle metadata", imageToFinalize, "for feature metadata.")
	devContainerJsonFeatureMap = convertLifecycleMetadataToFeatureOptionSelections(imageToFinalize, devContainerJsonFeatureMap)

	// Convert map back into a RawMessage for the map, and add it back into the devcontainer json object
	featureRawMessage, err := json.Marshal(devContainerJsonFeatureMap)
	if err != nil {
		log.Fatal(err)
	}
	devContainerJsonMap["features"] = featureRawMessage

	// If no devcontainer.json exists, create one with the image property set to the specified image
	targetDevContainerJsonPath := devContainerJsonPath
	if targetDevContainerJsonPath == "" {
		targetDevContainerJsonPath = filepath.Join(applicationFolder, ".devcontainer.json")
		devContainerJsonMap["image"] = []byte(imageToFinalize)
	}

	// Always disable command override, force user env probe
	devContainerJsonMap["overrideCommand"] = toJsonRawMessage(false)
	devContainerJsonMap["userEnvProbe"] = toJsonRawMessage("loginInteractiveShell")

	// Append ".buildpack" to avoid overwriting
	targetDevContainerJsonPath += ".buildpack"

	// Encode json content and write it to a file
	updatedDevContainerJsonContent, err := json.MarshalIndent(devContainerJsonMap, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Writing out updated devcontainer.json file:", targetDevContainerJsonPath)
	if err := WriteFile(targetDevContainerJsonPath, updatedDevContainerJsonContent); err != nil {
		log.Fatal(err)
	}

	log.Println("Ensuring /cnb/lifecycle/launch is fired as needed in bashrc/profile/zshenv.")
	updateImageToEnsureLauncherEnv(imageToFinalize)

	//log.Println("Calling devcontainer CLI to add remaining container features to image.")
	//devContainerImageBuild(imageToFinalize, targetDevContainerJsonPath, devContainerJsonPath)

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
		log.Fatal("Failed to build using devcontainer CLI. " + commandErr.Error())
	}
}

func updateImageToEnsureLauncherEnv(imageToFinalize string) {
	var err error
	tempDir := filepath.Join(os.TempDir(), strconv.FormatInt(rand.Int63(), 36))
	if err = os.MkdirAll(tempDir, 0777); err != nil {
		log.Fatal(err)
	}
	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err = WriteFile(dockerFilePath, envRestoreDockerfile); err != nil {
		log.Fatal(err)
	}
	ensureLauncherEnvScriptPath := filepath.Join(tempDir, "ensure-launcher-env.sh")
	if err = WriteFile(ensureLauncherEnvScriptPath, ensureLauncherEnvScript); err != nil {
		log.Fatal(err)
	}
	dockerCli(tempDir, false, "build", "--build-arg", "IMAGE_NAME="+imageToFinalize, "-t", imageToFinalize, "-f", dockerFilePath, ".")
	if err = os.RemoveAll(tempDir); err != nil {
		log.Fatal(err)
	}
}

func convertLifecycleMetadataToFeatureOptionSelections(imageToFinalize string, devContainerJsonFeatureMap map[string]interface{}) map[string]interface{} {
	// Inspect the specified image to get any feature config set on a label
	labelJsonBytes := dockerCli("", true, "image", "inspect", imageToFinalize, "-f", "{{ index .Config.Labels \"io.buildpacks.lifecycle.metadata\" }}")
	if len(labelJsonBytes) <= 0 {
		log.Println("No features metadata in image, so no post processing required.")
		return devContainerJsonFeatureMap
	}
	var lifecycleLabelData LifecycleLabelMetadata
	if err := json.Unmarshal(labelJsonBytes, &lifecycleLabelData); err != nil {
		log.Fatal(err)
	}
	// For each buildpack
	for _, buildpackMetadata := range lifecycleLabelData.Buildpacks {
		// And each layer in each buildpack
		for _, layerMetadata := range buildpackMetadata.Layers {
			// See if there is any feature metadata
			if layerFeatureMetadataRaw, hasKey := layerMetadata.Data[FeatureLayerMetadataId]; hasKey {
				// If so, load the json contents
				var layerFeatureMetadata LayerFeatureMetadata
				if err := json.Unmarshal(layerFeatureMetadataRaw, &layerFeatureMetadata); err != nil {
					log.Fatal(err)
				}
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
	return devContainerJsonFeatureMap
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
		log.Fatal("Docker command failed. " + errorOutput.String() + commandErr.Error())
	}
	return outputBytes.Bytes()
}

func toJsonRawMessage(value interface{}) json.RawMessage {
	var err error
	var bytes json.RawMessage
	if bytes, err = json.Marshal(value); err != nil {
		log.Fatal(err)
	}
	return bytes
}
