package main

import (
	"encoding/json"
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
	OptionSelections  map[string]string
}

// Implementation of libcnb.Builder.Build
func (fb FeatureBuilder) Build(context libcnb.BuildContext) (libcnb.BuildResult, error) {
	log.Println("Buildpack path:", context.Buildpack.Path)
	log.Println("Application path:", context.Application.Path)
	log.Println("Number of plan entries:", len(context.Plan.Entries))
	log.Println("Env:", os.Environ())

	var result libcnb.BuildResult

	// Load devcontainer.json, features.json, buildpack settings
	devContainerJson, _ := LoadDevContainerJson(context.Application.Path)
	buildpackSettings := LoadBuildpackSettings(context.Buildpack.Path)
	featuresJson := LoadFeaturesJson(context.Buildpack.Path)
	log.Println("Number of features in Buildpack:", len(featuresJson.Features))

	// Process each feature if it is in the buildpack plan in the order they appear in features.json
	for _, feature := range featuresJson.Features {
		optionSelections := GetOptionSelections(feature, buildpackSettings, devContainerJson)
		shouldAddLayer, layerContributor := getLayerContributorForFeature(feature, buildpackSettings, optionSelections, context.Plan)
		if shouldAddLayer {
			layerContributor.Context = context
			result.Layers = append(result.Layers, layerContributor)
			// TODO: Handle entrypoints? Or leave this to devcontainer CLI?
		}
	}
	// Generate any unmet entries
	for _, entry := range context.Plan.Entries {
		met := false
		for _, layer := range result.Layers {
			if entry.Name == layer.(FeatureLayerContributor).FullFeatureId() {
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

	// Write label metadata (from layers) to indicate which features should be processed by devcontainer CLI
	label := libcnb.Label{Key: AppliedFeaturesLabelId}
	allFeatureOptionSelections := make(map[string]map[string]string)
	for _, layer := range result.Layers {
		contributor := layer.(FeatureLayerContributor)
		idAndVersion := contributor.FullFeatureId() + "@" + buildpackSettings.Version
		if contributor.OptionSelections != nil {
			allFeatureOptionSelections[idAndVersion] = contributor.OptionSelections
		} else {
			allFeatureOptionSelections[idAndVersion] = make(map[string]string)
		}
	}
	labelBytes, err := json.Marshal(allFeatureOptionSelections)
	if err != nil {
		return result, err
	}
	label.Value = string(labelBytes)
	result.Labels = append(result.Labels, label)
	return result, nil
}

func (fc FeatureLayerContributor) FullFeatureId() string {
	return GetFullFeatureId(fc.Feature, fc.BuildpackSettings)
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
		log.Printf("- Skipping feature %s. No acquire script.", fc.FullFeatureId())
		return layer, nil
	}

	// Get build environment based on set options
	env := GetBuildEnvironment(fc.Feature, fc.OptionSelections, layer.Path)

	// Execute the script
	log.Printf("- Executing %s\n", acquireScriptPath)
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

	// Add ID and option selections to layer metadata, add to LayerContributor
	layer.Metadata = make(map[string]interface{})
	layer.Metadata["id"] = fc.FullFeatureId()
	for name, value := range fc.OptionSelections {
		layer.Metadata[name] = value
	}

	//TODO: Handle containerEnv? This only works if the buildpack entrypoint is used - problem for SSH
	if fc.Feature.ContainerEnv != nil && len(fc.Feature.ContainerEnv) > 0 {
		for name, value := range fc.Feature.ContainerEnv {
			before, after, overwrite := processEnvVar(name, value, fc.Feature.ContainerEnv)
			if before != "" || after != "" {
				layer.SharedEnvironment.Prepend(name, "", before)
				layer.SharedEnvironment.Append(name, "", after)
			} else {
				layer.SharedEnvironment.Override(name, overwrite)
			}
		}
	}

	// Finally, update layer types based on what was detected when created
	layer.LayerTypes = fc.LayerTypes

	return layer, nil
}

// See if the build plan includes an entry for this feature. If so, return a LayerContributor for it
func getLayerContributorForFeature(feature FeatureConfig, buildpackSettings BuildpackSettings, optionSelections map[string]string, plan libcnb.BuildpackPlan) (bool, FeatureLayerContributor) {
	layerContributor := FeatureLayerContributor{Feature: feature, BuildpackSettings: buildpackSettings, OptionSelections: optionSelections}
	// See if detect said should provide this feature
	for _, entry := range plan.Entries {
		// See if this entry is for this feature
		fullFeatureId := layerContributor.FullFeatureId()
		if entry.Name == fullFeatureId {
			log.Printf("- Entry for %s found", fullFeatureId)

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

func processEnvVar(name string, value string, envVars map[string]string) (string, string, string) {
	before := ""
	after := ""
	overwrite := ""

	// Handle self-referencing
	selfReplaceString := "${containerEnv:" + name + "}"
	selfRefIndex := strings.Index(value, selfReplaceString)
	if selfRefIndex > -1 {
		before = value[:selfRefIndex]
		after = value[selfRefIndex+len(selfReplaceString):]
	} else {
		overwrite = value
	}

	// Replace other variables set
	for otherVarName, otherVarValue := range envVars {
		if otherVarName != name {
			replaceString := "${containerEnv:" + otherVarName + "}"
			before = strings.ReplaceAll(before, replaceString, otherVarValue)
			after = strings.ReplaceAll(after, replaceString, otherVarValue)
			overwrite = strings.ReplaceAll(overwrite, replaceString, otherVarValue)
		}
	}

	return before, after, overwrite
}
