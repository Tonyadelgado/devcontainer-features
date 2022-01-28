package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
	if len(os.Args) < 3 {
		fmt.Println("Usage: build <layers-dir> <platform-dir> <path to Buildpack Plan toml file>")
		os.Exit(1)
	}
	layersDir := os.Args[1]
	platformDir := os.Args[2]
	envDir := filepath.Join(platformDir, "env")
	planPath := os.Args[3]

	log.Printf("Parameters:\n- Layers dir: %s\n- Platform dir: %s\n- Plan path: %s\n- CNB_BUILDPACK_DIR: %s\n", layersDir, platformDir, planPath, os.Getenv("CNB_BUILDPACK_DIR"))
	log.Println("Env:", os.Environ())

	// Load features.json, buildpack settings
	featuresJson := libbuildpackify.LoadFeaturesJson()
	buildpackSettings := libbuildpackify.LoadBuildpackSettings()

	// Load Buildpack Plan - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpack-plan-toml
	var plan libcnb.BuildpackPlan
	if _, err := toml.DecodeFile(planPath, &plan); err != nil {
		log.Fatal(err)
	}
	log.Printf("Plan entries: %d\n", len(plan.Entries))

	// Process each feature if it is in the buildpack plan in the order they appear in features.json
	var layers []libcnb.Layer
	for _, feature := range featuresJson.Features {
		layerAdded, layer := buildFeatureIfInPlan(buildpackSettings, feature, &plan, layersDir, envDir)
		if layerAdded {
			layers = append(layers, layer)
			// TODO: Handle entrypoints? Or leave this to devcontainer CLI?
		}
	}
	log.Printf("Number of layers added: %d", len(layers))

	// Write unmet dependencies in build.toml - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildtoml-toml
	// The buildFeatures method removes any unmet dependencies from the plan, so iterate through remaining entries.
	var buildToml libcnb.BuildTOML
	for _, entry := range plan.Entries {
		unmetEntry := libcnb.UnmetPlanEntry{Name: entry.Name}
		buildToml.Unmet = append(buildToml.Unmet, unmetEntry)
	}
	log.Printf("Unmet dependencies: %d", len(layers))
	file, err := os.OpenFile(filepath.Join(layersDir, "build.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(buildToml)

	// TODO: Write launch.toml with label metadata (from layers) to indicate which features should be processed by devcontainer CLI
}

func buildFeatureIfInPlan(buildpackSettings libbuildpackify.BuildpackSettings, feature libbuildpackify.FeatureConfig, plan *libcnb.BuildpackPlan, layersDir string, envDir string) (bool, libcnb.Layer) {
	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet + "/" + feature.Id
	fullFeatureIdWithDashes := buildpackSettings.Publisher + "-" + buildpackSettings.FeatureSet + "-" + feature.Id
	targetLayerPath := filepath.Join(layersDir, fullFeatureIdWithDashes)

	shouldCreateLayer, layer := updatePlanAndGetLayerForFeature(fullFeatureId, plan)
	if !shouldCreateLayer {
		return false, layer
	}

	// Check if acquire script for feature exists, exit otherwise
	acquireScriptPath := libbuildpackify.GetFeatureScriptPath(feature.Id, "acquire")
	_, err := os.Stat(acquireScriptPath)
	if err != nil {
		log.Printf("No acquire script for feature %s. Skipping.", fullFeatureId)
		return false, layer
	}

	// Create environment that includes feature build args
	idSafe := strings.ReplaceAll(strings.ToUpper(feature.Id), "-", "_")
	optionEnvVarPrefix := "_BUILD_ARG_" + idSafe
	env := append(os.Environ(),
		optionEnvVarPrefix+"=true",
		optionEnvVarPrefix+"_TARGET_PATH="+targetLayerPath)

	setOptions := make(map[string]string)

	// TODO: Inspect devcontainer.json if present to find options
	// TODO: Vary dev container verses feature processing somehow

	// Look for BP_DEV_CONTAINER_FEATURE_<feature.Id>_<option> environment variables, convert
	for optionName := range feature.Options {
		optionNameSafe := strings.ReplaceAll(strings.ToUpper(optionName), "-", "_")
		optionValue := os.Getenv("BP_DEV_CONTAINER_FEATURE_" + idSafe + "_" + optionNameSafe)
		if optionValue != "" {
			env = append(env, optionEnvVarPrefix+"_"+optionName+"=\""+optionValue+"\"")
			setOptions[optionName] = optionValue
		}
	}

	// TODO: Populate env directory with any env vars set in features.json?

	// Execute the script
	log.Printf("Executing %s\n", acquireScriptPath)
	logWriter := log.Writer()
	acquireCommand := exec.Command(acquireScriptPath)
	acquireCommand.Env = env
	acquireCommand.Stdout = logWriter
	acquireCommand.Stderr = logWriter

	if err := acquireCommand.Run(); err != nil {
		log.Fatal(err)
	}
	exitCode := acquireCommand.ProcessState.ExitCode()
	if exitCode != 0 {
		log.Printf("Error executing %s. Exit code %d.\n", acquireScriptPath, exitCode)
		os.Exit(exitCode)
	}

	// Add ID and options to layer metadata
	layer.Metadata = make(map[string]interface{})
	layer.Metadata["id"] = fullFeatureId
	for name, value := range setOptions {
		layer.Metadata[name] = value
	}

	// Write <full-feature-id>.toml - https://github.com/buildpacks/spec/blob/main/buildpack.md#layer-content-metadata-toml
	file, err := os.OpenFile(filepath.Join(layersDir, fullFeatureIdWithDashes+".toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(layer)

	return true, layer
}

// See if the build plan includes an entry for this feature. If so, remove it
// from the plan and return the layer types for use in this buildpack
func updatePlanAndGetLayerForFeature(fullFeatureId string, plan *libcnb.BuildpackPlan) (bool, libcnb.Layer) {
	var layer libcnb.Layer
	// See if detect said should provide this feature
	for i := len(plan.Entries) - 1; i >= 0; i-- {
		entry := plan.Entries[i]
		if entry.Name == fullFeatureId {
			log.Printf("Found entry for %s", fullFeatureId)
			// Remove this entry from the plan
			if len(plan.Entries) > 1 {
				copy(plan.Entries[:i], plan.Entries[i+1:])
			} else {
				plan.Entries = []libcnb.BuildpackPlanEntry{}
			}
			// Set layer types
			var layerTypes libcnb.LayerTypes
			for _, key := range []string{"Build", "Launch", "Cache"} {
				// If entry metadata contains the build, Launch, or cache keys, set
				// it on the LayerTypes object using reflection, otherwise set to true
				value, containsKey := entry.Metadata[strings.ToLower(key)]
				field := reflect.ValueOf(&layerTypes).Elem().FieldByName(key)
				if containsKey {
					field.Set(reflect.ValueOf(value.(bool)))
				} else {
					// default is true
					field.Set(reflect.ValueOf(true))
				}
			}
			layer.LayerTypes = layerTypes

			// TODO: Verify whether multiple entries for the same feature could theoretically be present

			return true, layer
		}
	}

	return false, layer
}
