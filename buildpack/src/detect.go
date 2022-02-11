package main

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/buildpacks/libcnb"
)

type FeatureDetector struct {
	// Implements libcnb.Detector
	// Detect(context libcnb.DetectContext) (libcnb.DetectResult, error)
}

// Implementation of libcnb.Detector.Detect
func (fd FeatureDetector) Detect(context libcnb.DetectContext) (libcnb.DetectResult, error) {
	log.Println("Buildpack path:", context.Buildpack.Path)
	log.Println("Application path:", context.Application.Path)
	log.Println("Env:", os.Environ())

	var result libcnb.DetectResult

	// Load features.json, buildpack settings
	devContainerJson, _ := LoadDevContainerJson(context.Application.Path)
	buildpackSettings := LoadBuildpackSettings(context.Buildpack.Path)
	featuresJson := LoadFeaturesJson(context.Buildpack.Path)
	log.Println("Number of features in Buildpack:", len(featuresJson.Features))

	// See if should provide any features
	var plan libcnb.BuildPlan
	onlyProvided := []libcnb.BuildPlanProvide{}
	for _, feature := range featuresJson.Features {
		detected, provide, require, err := detectFeature(context, buildpackSettings, feature, devContainerJson)
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

	result.Pass = true
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
		log.Println(optionalPlan.Provides)
		result.Plans = append(result.Plans, optionalPlan)
	}

	return result, nil
}

func detectFeature(context libcnb.DetectContext, buildpackSettings BuildpackSettings, feature FeatureConfig, devContainerJson DevContainerJson) (bool, libcnb.BuildPlanProvide, libcnb.BuildPlanRequire, error) {
	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := GetFullFeatureId(feature, buildpackSettings, "/")
	provide := libcnb.BuildPlanProvide{Name: fullFeatureId}
	require := libcnb.BuildPlanRequire{Name: fullFeatureId}

	// Check environment to see if BP_CONTAINER_FEATURE_<feature.Id> is set
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	if os.Getenv("BP_CONTAINER_FEATURE_"+idSafe) != "" {
		return true, provide, require, nil
	}

	// If we're in devcontainer mode, and its referenced in devcontainer.json, return true
	if GetContainerImageBuildMode() == "devcontainer" {
		for featureName := range devContainerJson.Features {
			if featureName == fullFeatureId || strings.HasPrefix(featureName, fullFeatureId+"@") {
				return true, provide, require, nil
			}
		}
	}

	// Otherwise, check if detect script for feature exists, return not detected otherwise
	detectScriptPath := GetFeatureScriptPath(context.Buildpack.Path, feature.Id, "detect")
	_, err := os.Stat(detectScriptPath)
	if err != nil {
		return false, provide, require, nil
	}

	// Execute the script
	log.Printf("- Executing %s\n", detectScriptPath)
	optionSelections := GetOptionSelections(feature, buildpackSettings, devContainerJson)
	env := GetBuildEnvironment(feature, optionSelections)
	logWriter := log.Writer()
	detectCommand := exec.Command(detectScriptPath)
	detectCommand.Env = env
	detectCommand.Stdout = logWriter
	detectCommand.Stderr = logWriter
	if err := detectCommand.Run(); err != nil {
		log.Fatal(err)
	}

	exitCode := detectCommand.ProcessState.ExitCode()
	switch exitCode {
	case 0: // Detected
		return true, provide, require, nil
	case 100: // Not detected
		return false, provide, require, nil
	default:
		return false, provide, require, NonZeroExitError{ExitCode: exitCode}
	}
}
