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

type LayerFeatureMetadata struct {
	Id         string
	Version    string
	Selections map[string]string
}

type LabelBuldpack struct {
	Layers map[string]LabelBuldpackLayer
}

type LabelBuldpackLayer struct {
	Data map[string]json.RawMessage
}

type LifecycleLabelMetadata struct {
	Buildpacks []LabelBuldpack
}

// docker inspect test_image -f '{{ index .Config.Labels "io.buildpacks.lifecycle.metadata" }}'
//go:embed assets/extract-buildpack-env.sh
var envVarExtractionScript []byte

//go:embed assets/env-restore.Dockerfile
var envRestoreDockerfile []byte

func FinalizeImage(imageToFinalize string, applicationFolder string, buildMode string) {
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

	extractAndMakeEnvVarsGlobal(imageToFinalize)
}

func extractAndMakeEnvVarsGlobal(imageToFinalize string) {
	var err error
	log.Println("Extracting env vars from", imageToFinalize, "for default launch target.")
	tempDir := filepath.Join(os.TempDir(), strconv.FormatInt(rand.Int63(), 36))
	if err = os.MkdirAll(tempDir, 0777); err != nil {
		log.Fatal(err)
	}
	envDockerfileSnippetBytes := dockerCli(tempDir, true, "run", "--rm", imageToFinalize, string(envVarExtractionScript))
	envFilePath := filepath.Join(tempDir, "buildpack.env")
	if err = WriteFile(envFilePath, envDockerfileSnippetBytes); err != nil {
		log.Fatal(err)
	}

	dockerFilePath := filepath.Join(tempDir, "Dockerfile")
	if err = WriteFile(dockerFilePath, envRestoreDockerfile); err != nil {
		log.Fatal(err)
	}
	dockerCli(tempDir, false, "build", "--build-arg", "IMAGE_NAME="+imageToFinalize, "-t", imageToFinalize, "-f", dockerFilePath, ".")
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
				devContainerJsonFeatureMap[layerFeatureMetadata.Id+"@"+layerFeatureMetadata.Version] = layerFeatureMetadata.Selections
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
		log.Fatal("Failed to extract env vars from Docker image. " + errorOutput.String() + commandErr.Error())
	}
	return outputBytes.Bytes()
}
