package main

import (
	"log"
	"os"
	"os/exec"
	"reflect"
	"strings"

	"github.com/buildpacks/libcnb"
)

type FeatureBuilder struct {
	// Implements libcnb.Builder
	// Build(context libcnb.BuildContext) (libcnb.BuildResult, error)
}

type FeatureLayerContributor struct {
	// Implements libcnb.LayerContributor
	// Contribute(context libcnb.ContributeContext) (libcnb.Layer, error)
	// Name() string

	// FullFeatureId() string
	Feature           FeatureConfig
	BuildpackSettings BuildpackSettings
	LayerTypes        libcnb.LayerTypes
	Context           libcnb.BuildContext
}

// Implementation of libcnb.Builder.Build
func (fb FeatureBuilder) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	log.Println("Context: ", context)
	log.Println("Env: ", os.Environ())

	var result libcnb.BuildResult

	// Load features.json, buildpack settings
	featuresJson := LoadFeaturesJson(context.Buildpack.Path)
	buildpackSettings := LoadBuildpackSettings(context.Buildpack.Path)

	// Load Buildpack Plan - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpack-plan-toml
	log.Printf("Plan entries: %d\n", len(context.Plan.Entries))

	// Process each feature if it is in the buildpack plan in the order they appear in features.json
	var metEntries []string
	for _, feature := range featuresJson.Features {
		shouldAddLayer, layerContributor := getLayerContributorForFeature(feature, buildpackSettings, context.Plan)
		if shouldAddLayer {
			layerContributor.Context = context
			metEntries = append(metEntries, layerContributor.FullFeatureId())
			result.Layers = append(result.Layers, layerContributor)
			// TODO: Handle entrypoints? Or leave this to devcontainer CLI?
		}
	}
	// Generate any unmet entries
	for _, entry := range context.Plan.Entries {
		met := false
		for _, metEntry := range metEntries {
			if entry.Name == metEntry {
				met = true
				break
			}
		}
		if !met {
			result.Unmet = append(result.Unmet, libcnb.UnmetPlanEntry{Name: entry.Name})
		}
	}
	log.Printf("Number of layer contributors: %d", len(result.Layers))
	log.Printf("Unmet entries: %d", len(result.Unmet))

	// TODO: Write launch.toml with label metadata (from layers) to indicate which features should be processed by devcontainer CLI
	return result, nil
}

func (fc FeatureLayerContributor) FullFeatureId() string {
	// e.g. chuxel/devcontainer/features/packcli
	return fc.BuildpackSettings.Publisher + "/" + fc.BuildpackSettings.FeatureSet + "/" + fc.Feature.Id
}

// Implementation of libcnb.LayerContributor.Name
func (fc FeatureLayerContributor) Name() string {
	// e.g. chuxel-devcontainer-features-packcli
	return fc.BuildpackSettings.Publisher + "-" + fc.BuildpackSettings.FeatureSet + "-" + fc.Feature.Id
}

// Implementation of libcnb.LayerContributor.Contribute
func (fc FeatureLayerContributor) Contribute(layer libcnb.Layer) (libcnb.Layer, error) {
	var err error
	// Check if acquire script for feature exists, exit otherwise
	acquireScriptPath := GetFeatureScriptPath(fc.Context.Buildpack.Path, fc.Feature.Id, "acquire")
	_, err = os.Stat(acquireScriptPath)
	if err != nil {
		log.Printf("No acquire script for feature %s. Skipping.", fc.FullFeatureId())
		return layer, nil
	}

	// Get build environment based on set options
	env, setOptions := GetBuildEnvironment(fc.Feature, layer.Path)

	// Execute the script
	log.Printf("Executing %s\n", acquireScriptPath)
	logWriter := log.Writer()
	acquireCommand := exec.Command(acquireScriptPath)
	acquireCommand.Env = env
	acquireCommand.Stdout = logWriter
	acquireCommand.Stderr = logWriter
	acquireCommand.Dir = fc.Context.Application.Path

	if err := acquireCommand.Run(); err != nil {
		return layer, err
	}
	exitCode := acquireCommand.ProcessState.ExitCode()
	if exitCode != 0 {
		log.Printf("Error executing %s. Exit code %d.\n", acquireScriptPath, exitCode)
		return layer, NonZeroExitError{ExitCode: exitCode}
	}

	// Add ID and options to layer metadata
	layer.Metadata = make(map[string]interface{})
	layer.Metadata["id"] = fc.FullFeatureId()
	for name, value := range setOptions {
		layer.Metadata[name] = value
	}

	//TODO: Handle containerEnv?
	/* Since containerEnv can contain values that reference other variables,
		leave processing to after the script has executed already.

	// Add any containerEnv values to layer specific env file
	if fc.Feature.ContainerEnv != nil && len(fc.Feature.ContainerEnv) > 0 {
		for name, value := range fc.Feature.ContainerEnv {
			// Swap out "containerEnv" values with normal variable strings
			formattedValue := strings.ReplaceAll(value, "${containerEnv:", "${")
			layer.SharedEnvironment[name] = os.ExpandEnv(formattedValue)
		}
	}
	*/

	// Finally, update layer types based on what was detected when created
	layer.LayerTypes = fc.LayerTypes

	return layer, nil
}

// See if the build plan includes an entry for this feature. If so, return a LayerContributor for it
func getLayerContributorForFeature(feature FeatureConfig, buildpackSettings BuildpackSettings, plan libcnb.BuildpackPlan) (bool, FeatureLayerContributor) {
	layerContributor := FeatureLayerContributor{Feature: feature, BuildpackSettings: buildpackSettings}
	// See if detect said should provide this feature
	for _, entry := range plan.Entries {
		// See if this entry is for this feature
		fullFeatureId := layerContributor.FullFeatureId()
		if entry.Name == fullFeatureId {
			log.Printf("Found entry for %s", fullFeatureId)

			// If entry metadata contains the build, Launch, or cache keys, set
			// it on the LayerTypes object using reflection, otherwise set to true
			var layerTypes libcnb.LayerTypes
			for _, key := range []string{"Build", "Launch", "Cache"} {
				value, containsKey := entry.Metadata[strings.ToLower(key)]
				field := reflect.ValueOf(&layerTypes).Elem().FieldByName(key)
				if containsKey {
					field.Set(reflect.ValueOf(value.(bool)))
				} else {
					// default is true
					field.Set(reflect.ValueOf(true))
				}
			}
			layerContributor.LayerTypes = layerTypes

			return true, layerContributor
		}
	}

	return false, layerContributor
}
