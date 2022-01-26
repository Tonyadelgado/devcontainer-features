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

	// Load features.json, buildpack settings
	featuresJson := libbuildpackify.LoadFeaturesJson()
	buildpackSettings := libbuildpackify.LoadBuildpackSettings()

	// Load Buildpack Plan - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpack-plan-toml
	var plan libcnb.BuildpackPlan
	if _, err := toml.DecodeFile(planPath, &plan); err != nil {
		log.Fatal(err)
	}
	log.Printf("Plan entries: %d", len(plan.Entries))

	// Process each feature if it is in the buildpack plan in the order they appear in features.json
	var layers []libcnb.Layer
	for _, feature := range featuresJson.Features {
		layerAdded, layer := buildFeatureIfInPlan(buildpackSettings, feature, &plan, layersDir, envDir)
		if layerAdded {
			layers = append(layers, layer)
		}
	}

	// Write unmet dependencies in build.toml - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildtoml-toml
	// The buildFeatures method removes any unmet dependencies from the plan, so iterate through remaining entries.
	var buildToml libcnb.BuildTOML
	for _, entry := range plan.Entries {
		unmetEntry := libcnb.UnmetPlanEntry{Name: entry.Name}
		buildToml.Unmet = append(buildToml.Unmet, unmetEntry)
	}
	file, err := os.OpenFile(filepath.Join(layersDir, "build.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(buildToml)

	// TODO: Write launch.toml with label metadata to indicate build was executed based on metadata in layers
	log.Printf("Number of layers added: %d", len(layers))
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
		return false, layer
	}

	// Create environment that includes feature build args
	argPrefix := "_BUILD_ARG_" + strings.ToUpper(feature.Id)
	env := append(os.Environ(),
		argPrefix+"=true",
		argPrefix+"_TARGET_PATH="+targetLayerPath)
	// TODO: Inspect devcontainer.json if present to find options, generate args, add these as labels

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
	// TODO: Check exit code

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
	shouldCreateLayer := false
	// See if detect said should provide this feature
	for i := len(plan.Entries) - 1; i >= 0; i-- {
		entry := plan.Entries[i]
		if entry.Name == fullFeatureId {
			shouldCreateLayer = true
			// Remove this entry from the plan
			copy(plan.Entries[i:], plan.Entries[i+1:])
			plan.Entries = plan.Entries[:len(plan.Entries)-1]
			plan.Entries = plan.Entries[:i]

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
			break
		}
	}

	return shouldCreateLayer, layer
}
