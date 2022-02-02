package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func FinalizeImage(imageToFinalize string, applicationFolder string, context string) {
	log.Println("Image to finalize:", imageToFinalize)
	log.Println("Application folder:", applicationFolder)
	log.Println("Container image build context:", context)

	// Get devcontainer.json as a map so we don't change any fields unexpectedly
	var devContainerJsonMap map[string]json.RawMessage
	var devContainerJsonFeatureMap map[string]interface{}
	var devContainerJsonPath string
	// Only load the existing devcontainer.json file if we're in the devcontainer context
	if context == "devcontainer" {
		log.Println("Loading devcontainer.json if present.")
		devContainerJsonMap, devContainerJsonPath = LoadDevContainerJsonAsMap(applicationFolder)
		// Unmarshall devcontainer.json features into a map
		if err := json.Unmarshal(devContainerJsonMap["features"], &devContainerJsonFeatureMap); err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Context is", context, "- skipping devcontainer.json load.")
		devContainerJsonPath = FindDevContainerJson(applicationFolder)
	}

	// Inspect the specified image to get any feature config set on a label
	log.Println("Inspecting container image", imageToFinalize, "for feature metadata.")
	var labelJsonBytes bytes.Buffer
	var errorOutput bytes.Buffer
	dockerCommand := exec.Command("docker", "image", "inspect", imageToFinalize, "-f", "{{ index .Config.Labels \""+AppliedFeaturesLabelId+"\" }}")
	dockerCommand.Env = os.Environ()
	dockerCommand.Stdout = &labelJsonBytes
	dockerCommand.Stderr = &errorOutput
	commandErr := dockerCommand.Run()
	if commandErr != nil || dockerCommand.ProcessState.ExitCode() != 0 || errorOutput.Len() != 0 {
		log.Fatal("Failed to inspect Docker image. " + errorOutput.String() + commandErr.Error())
	}
	if labelJsonBytes.Len() > 0 {
		// Unmarshall the result if we got anything
		var labelFeatureMap map[string]interface{}
		if err := json.Unmarshal(labelJsonBytes.Bytes(), &labelFeatureMap); err != nil {
			log.Fatal(err)
		}
		for labelFeatureId, labelOptionSelections := range labelFeatureMap {
			// Remove existing values in devcontainer.json if any are found so we can set them cleanly w/o worrying about versions
			labelIdWOVersion := strings.Split(labelFeatureId, "@")[0]
			for featureId := range devContainerJsonFeatureMap {
				if featureId == labelFeatureId || featureId == labelIdWOVersion || strings.HasPrefix(featureId, labelIdWOVersion+"@") {
					delete(devContainerJsonFeatureMap, featureId)
					break
				}
			}
			devContainerJsonFeatureMap[labelFeatureId] = labelOptionSelections
		}

		// Convert map back into a RawMessage for the map
		featureRawMessage, err := json.Marshal(devContainerJsonFeatureMap)
		if err != nil {
			log.Fatal(err)
		}
		devContainerJsonMap["features"] = featureRawMessage
	} else {
		log.Println("No features metadata in image, so no post processing required.")
	}

	// If no devcontainer.json exists, create one with the image property set to the specified image
	targetDevContainerJsonPath := devContainerJsonPath
	if targetDevContainerJsonPath == "" {
		targetDevContainerJsonPath = filepath.Join(applicationFolder, ".devcontainer.json")
		devContainerJsonMap["image"] = []byte(imageToFinalize)
	}
	// Append ".build" to avoid overwriting
	targetDevContainerJsonPath += ".build"

	// Encode json content and write it to a file
	updatedDevContainerJsonContent, err := json.MarshalIndent(devContainerJsonMap, "", "\t")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Writing out updated devcontainer.json file:", targetDevContainerJsonPath)
	if err := WriteFile(targetDevContainerJsonPath, updatedDevContainerJsonContent); err != nil {
		log.Fatal(err)
	}

}
