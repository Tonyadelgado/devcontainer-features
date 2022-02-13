package main

import (
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/joho/godotenv"
)

const DevContainerFeaturesEnvPath = "/tmp/devcontainer-features.env"

type FeatureDetector struct {
	// Implements libcnb.Detector
	// Detect(context libcnb.DetectContext) (libcnb.DetectResult, error)
}

// Implementation of libcnb.Detector.Detect
func (fd FeatureDetector) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	log.Println("Devpack path:", context.Buildpack.Path)
	log.Println("Application path:", context.Application.Path)
	log.Println("Env:", os.Environ())

	var result libcnb.DetectResult

	// TODO: Enable detect script to return options that should then be passed on to the builder via metadata, then drop devcontainer.json parsing in the build stage

	// Load features.json, buildpack settings
	devpackSettings := LoadDevpackSettings(context.Buildpack.Path)
	featuresJson := LoadFeaturesJson(context.Buildpack.Path)
	log.Println("Number of features in Devpack:", len(featuresJson.Features))

	// Load devcontainer.json if in devcontainer build mode
	var devContainerJson DevContainerJson
	if GetContainerImageBuildMode() == "devcontainer" {
		devContainerJson, _ = LoadDevContainerJson(context.Application.Path)
	}

	// See if should provide any features
	var plan libcnb.BuildPlan
	onlyProvided := []libcnb.BuildPlanProvide{}
	for _, feature := range featuresJson.Features {
		detected, provide, require, err := detectFeature(context, devpackSettings, feature, devContainerJson)
		if err != nil {
			return result, err
		}
		if detected {
			log.Printf("- %s detected\n", feature.Id)
			plan.Provides = append(plan.Provides, provide)
			plan.Requires = append(plan.Requires, require)
		} else {
			onlyProvided = append(onlyProvided, provide)
			log.Printf("- %s provided\n", provide.Name)
		}

	}

	result.Plans = append(result.Plans, plan)
	// Generate all permutations where something is just provided
	combinationList := GetAllCombinations(len(onlyProvided))
	for _, combination := range combinationList {
		var optionalPlan libcnb.BuildPlan
		copy(optionalPlan.Requires, plan.Requires)
		copy(optionalPlan.Provides, plan.Provides)
		for _, i := range combination {
			optionalPlan.Provides = append(optionalPlan.Provides, onlyProvided[i])
		}
		result.Plans = append(result.Plans, optionalPlan)
	}

	// Always pass since we can provide features even if they're not used by this buildpack
	result.Pass = true
	return result, nil
}

func detectFeature(context libcnb.DetectContext, buildpackSettings DevpackSettings, feature FeatureConfig, devContainerJson DevContainerJson) (bool, libcnb.BuildPlanProvide, libcnb.BuildPlanRequire, error) {
	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := GetFullFeatureId(feature, buildpackSettings, "/")
	provide := libcnb.BuildPlanProvide{Name: fullFeatureId}
	require := libcnb.BuildPlanRequire{Name: fullFeatureId, Metadata: make(map[string]interface{})}

	// Add any option selections from BP_CONTAINER_FEATURE_<feature.Id>_<option> env vars and devcontainer.json (in devcontainer mode)
	detected, optionSelections := detectOptionSelections(feature, buildpackSettings, devContainerJson)
	if detected {
		for optionId, selection := range optionSelections {
			require.Metadata[GetOptionMetadataKey(optionId)] = selection
		}
		return true, provide, require, nil
	}

	// Otherwise, check if detect script for feature exists, return not detected otherwise
	detectScriptPath := GetFeatureScriptPath(context.Buildpack.Path, feature.Id, "detect")
	_, err := os.Stat(detectScriptPath)
	if err != nil {
		return false, provide, require, nil
	}

	// Execute the script - set path to where a resulting devcontainer-features.env should be placed as env var
	log.Printf("- Executing %s\n", detectScriptPath)
	env := GetBuildEnvironment(feature, optionSelections, map[string]string{
		"SELECTION_ENV_FILE_PATH": DevContainerFeaturesEnvPath,
	})
	logWriter := log.Writer()
	detectCommand := exec.Command(detectScriptPath)
	detectCommand.Env = env
	detectCommand.Stdout = logWriter
	detectCommand.Stderr = logWriter
	if err := detectCommand.Run(); err != nil {
		log.Fatal(err)
	}

	exitCode := detectCommand.ProcessState.ExitCode()
	if exitCode == 0 {
		// Read option selections if any are provided
		if _, err := os.Stat(DevContainerFeaturesEnvPath); err != nil {
			if err := godotenv.Load(DevContainerFeaturesEnvPath); err != nil {
				log.Fatal(err)
			}
			_, optionSelections = mergeOptionSelectionsFromEnv(feature, optionSelections, OptionSelectionEnvVarPrefix)
			for option, selection := range optionSelections {
				require.Metadata[GetOptionMetadataKey(option)] = selection
			}
		}
		return true, provide, require, nil
	}
	// 100 means failed, other error codes mean an error ocurred
	if exitCode == 100 {
		return false, provide, require, nil
	} else {
		return false, provide, require, NonZeroExitError{ExitCode: exitCode}
	}
}

func detectOptionSelections(feature FeatureConfig, buildpackSettings DevpackSettings, devContainerJson DevContainerJson) (bool, map[string]string) {
	optionSelections := make(map[string]string)
	detectedDevContainerJson := false
	// If in dev container mode, parse devcontainer.json features (if any)
	if GetContainerImageBuildMode() == "devcontainer" {
		fullFeatureId := GetFullFeatureId(feature, buildpackSettings, "/")
		for featureName, jsonOptionSelections := range devContainerJson.Features {
			log.Println(featureName, "=", jsonOptionSelections)
			if featureName == fullFeatureId || strings.HasPrefix(featureName, fullFeatureId+"@") {
				detectedDevContainerJson = true
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
	detectedEnv, optionselections := mergeOptionSelectionsFromEnv(feature, optionSelections, ProjectTomlOptionSelectionEnvVarPrefix)
	return (detectedDevContainerJson || detectedEnv), optionselections
}

func mergeOptionSelectionsFromEnv(feature FeatureConfig, optionSelections map[string]string, prefix string) (bool, map[string]string) {
	detected := false
	enabledEnvVarVal := os.Getenv(GetOptionEnvVarName(prefix, feature.Id, ""))
	if enabledEnvVarVal != "" && enabledEnvVarVal != "false" {
		detected = true
	}
	for optionId := range feature.Options {
		optionValue := os.Getenv(GetOptionEnvVarName(prefix, feature.Id, optionId))
		if optionValue != "" {
			optionSelections[optionId] = optionValue
		}
	}
	return detected, optionSelections
}
