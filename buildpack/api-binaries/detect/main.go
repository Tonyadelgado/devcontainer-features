package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/chuxel/buildpackify-features/libbuildpackify"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
)

func main() {
	// Check that the buildpack location env var is set per https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpack-specific-variables
	if os.Getenv("CNB_BUILDPACK_DIR") == "" {
		fmt.Println("CNB_BUILDPACK_DIR environment variable not set. This should point to the location that\ncontains features.json, buildpack-features.json, and the features folder.")
		os.Exit(1)
	}
	// Build arguments, working directory expected to match https://github.com/buildpacks/spec/blob/main/buildpack.md#build
	if len(os.Args) < 2 {
		fmt.Println("Usage: detect  <platform-dir> <path to plan.toml file>")
		os.Exit(1)
	}
	platformDir := os.Args[1]
	planPath := os.Args[2]

	log.Printf("Parameters:\n- Platform dir: %s\n- Plan path: %s\n- CNB_BUILDPACK_DIR: %s\n", platformDir, planPath, os.Getenv("CNB_BUILDPACK_DIR"))
	log.Println("Env:", os.Environ())
	// Load features.json, buildpack settings
	featuresJson := libbuildpackify.LoadFeaturesJson()
	buildpackSettings := libbuildpackify.LoadBuildpackSettings()

	// Load build plan.toml
	var plan libcnb.BuildPlan
	if _, err := toml.DecodeFile(planPath, &plan); err != nil {
		log.Println("No existing plan.toml found.")
	}

	// See if should provide any features
	for _, feature := range featuresJson.Features {
		detected, provide, require := detectFeature(buildpackSettings, feature)
		if detected {
			log.Printf("Feature %s detected\n", feature.Id)
			plan.Provides = append(plan.Provides, provide)
			plan.Requires = append(plan.Requires, require)
		}
	}

	// Nothing detected
	if len(plan.Provides) == 0 {
		os.Exit(100)
	}

	// Write plan.toml
	file, err := os.OpenFile(planPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(plan)
}

func detectFeature(buildpackSettings libbuildpackify.BuildpackSettings, feature libbuildpackify.FeatureConfig) (bool, libcnb.BuildPlanProvide, libcnb.BuildPlanRequire) {
	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet + "/" + feature.Id
	provide := libcnb.BuildPlanProvide{Name: fullFeatureId}
	require := libcnb.BuildPlanRequire{Name: fullFeatureId}

	// TODO: Check devcontainer.json automatically in addition to firing detect if present

	// TODO: Check environment to see if BP_DEV_CONTAINER_FEATURE_<feature.Id> is set
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	if os.Getenv("BP_CONTAINER_FEATURE_"+idSafe) != "" {
		return true, provide, require
	}

	// Check if acquire script for feature exists, exit otherwise
	detectScriptPath := libbuildpackify.GetFeatureScriptPath(feature.Id, "detect")
	_, err := os.Stat(detectScriptPath)
	if err != nil {
		return false, provide, require
	}

	// Execute the script
	log.Printf("Executing %s\n", detectScriptPath)
	env, _ := libbuildpackify.GetBuildEnvironment(feature, "")
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
		return true, provide, require
	case 100: // Not detected
		return false, provide, require
	default:
		log.Printf("Error executing %s. Exit code %d.\n", detectScriptPath, exitCode)
		os.Exit(exitCode)
	}

	return false, provide, require
}
