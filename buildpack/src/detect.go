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
	log.Println("Context: ", context)
	log.Println("Env: ", os.Environ())

	var result libcnb.DetectResult

	// Load features.json, buildpack settings
	featuresJson := LoadFeaturesJson(context.Buildpack.Path)
	buildpackSettings := LoadBuildpackSettings(context.Buildpack.Path)

	// See if should provide any features
	var plan libcnb.BuildPlan
	for _, feature := range featuresJson.Features {
		detected, provide, require, err := detectFeature(context, buildpackSettings, feature)
		if err != nil {
			return result, err
		}
		if detected {
			log.Printf("Feature %s detected\n", feature.Id)
			plan.Provides = append(plan.Provides, provide)
			plan.Requires = append(plan.Requires, require)
		}
	}

	// Nothing detected
	if len(plan.Provides) == 0 {
		result.Pass = false
	}

	result.Pass = true
	result.Plans = append(result.Plans, plan)
	return result, nil
}

func detectFeature(context libcnb.DetectContext, buildpackSettings BuildpackSettings, feature FeatureConfig) (bool, libcnb.BuildPlanProvide, libcnb.BuildPlanRequire, error) {
	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet + "/" + feature.Id
	provide := libcnb.BuildPlanProvide{Name: fullFeatureId}
	require := libcnb.BuildPlanRequire{Name: fullFeatureId}

	// TODO: Check devcontainer.json automatically in addition to firing detect if present

	// Check environment to see if BP_CONTAINER_FEATURE_<feature.Id> is set
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	if os.Getenv("BP_CONTAINER_FEATURE_"+idSafe) != "" {
		return true, provide, require, nil
	}

	// Check if acquire script for feature exists, skip otherwise
	detectScriptPath := GetFeatureScriptPath(context.Buildpack.Path, feature.Id, "detect")
	_, err := os.Stat(detectScriptPath)
	if err != nil {
		return false, provide, require, nil
	}

	// Execute the script
	log.Printf("Executing %s\n", detectScriptPath)
	env, _ := GetBuildEnvironment(feature, "")
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