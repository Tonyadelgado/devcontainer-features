package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"syscall"

	"github.com/chuxel/buildpackify-features/libbuildpackify"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
)

func main() {
	layersDir := os.Args[1]
	envDir := filepath.Join(os.Args[2], "env")
	planPath := os.Args[3]

	// Load features.json, buildpack settings
	featuresJson := libbuildpackify.LoadFeaturesJson()
	buildpackSettings := libbuildpackify.LoadBuildpackSettings()

	// Load Buildpack Plan
	var plan libcnb.BuildpackPlan
	if _, err := toml.DecodeFile(planPath, &plan); err != nil {
		log.Fatal(err)
	}

	// Process each feature
	for _, feature := range featuresJson.Features {
		layerAdded, labelData := buildFeature(buildpackSettings, feature, &plan, layersDir, envDir)
		if layerAdded {
			// TODO: Copy over settings used to do the build into a structure to populate launch.toml with labels
			fmt.Printf("%s\n", labelData)
		}
	}

	// Write the updated build.toml
	file, err := os.OpenFile(filepath.Join(layersDir, "build.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(plan)

	// TODO: Write
}

func buildFeature(buildpackSettings libbuildpackify.BuildpackSettings, feature libbuildpackify.FeatureConfig, plan *libcnb.BuildpackPlan, layersDir string, envDir string) (bool, map[string]string) {
	// Get path to features content
	featuresPath := filepath.Join(os.Getenv("CNB_BUILDPACK_DIR"), "features")

	// e.g. chuxel/devcontainer/features/packcli
	fullFeatureId := buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet + "/" + feature.Id
	fullFeatureIdWithDashes := buildpackSettings.Publisher + "-" + buildpackSettings.FeatureSet + "-" + feature.Id
	targetLayerPath := filepath.Join(layersDir, fullFeatureIdWithDashes)

	var layerLabels map[string]string

	shouldCreateLayer, layer := updatePlanAndGetLayerForFeature(fullFeatureId, plan)
	if !shouldCreateLayer {
		return false, layerLabels
	}

	// Execute acquire script if present
	acquireScriptPath := filepath.Join(featuresPath, feature.Id, "bin", "acquire")
	_, err := os.Stat(acquireScriptPath)
	if err != nil {
		env := os.Environ()
		env = append(env, "_BUILD_ARG_"+feature.Id+"=true")
		env = append(env, "_BUILD_ARG_"+feature.Id+"_TARGET_PATH="+targetLayerPath)
		// TODO: Inspect devcontainer.json if present to find options

		// Current working directory is app directory per buildpack spec, so we can just execute
		syscall.Exec(acquireScriptPath, []string{}, env)
		// Write <full-feature-id>.toml
		file, err := os.OpenFile(filepath.Join(layersDir, fullFeatureIdWithDashes+".toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		toml.NewEncoder(file).Encode(layer)

		// TODO: Write launch.toml with label metadata to indicate feature was executed

	}

	return true, layerLabels
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
				// If entry metadata contains the Build, Launch, or cache keys, set
				// it on the LayerTypes object using reflection, otherwise set to true
				value, containsKey := entry.Metadata[key]
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
