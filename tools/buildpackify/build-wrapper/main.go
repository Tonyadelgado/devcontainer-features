package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
)

type FeatureMount struct {
	Source string
	Target string
	Type   string
}

type FeatureConfig struct {
	Id           string
	Name         string
	Options      map[string]string
	Entrypoint   string
	Privileged   bool
	Init         bool
	ContainerEnv []string
	Mounts       []FeatureMount
	CapAdd       []string
	SecurityOpt  []string
	BuildArg     string
}

type FeaturesJson struct {
	Features []FeatureConfig
}

func loadFeaturesJson(featuresPath string) FeaturesJson {
	// Load features.json
	content, err := ioutil.ReadFile(filepath.Join(featuresPath, "features.json"))
	if err != nil {
		log.Fatal(err)
	}
	var featuresJson FeaturesJson
	err = json.Unmarshal(content, &featuresJson)
	if err != nil {
		log.Fatal(err)
	}

	return featuresJson
}

func main() {
	layersDir := os.Args[1]
	envDir := filepath.Join(os.Args[2], "env")
	planPath := os.Args[3]

	// Get path to features content
	binaryFilePath, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	featuresPath := filepath.Join(filepath.Dir(binaryFilePath), "..", "features")

	// Load features.json
	featuresJson := loadFeaturesJson(featuresPath)

	// Load Buildpack Plan
	var plan libcnb.BuildpackPlan
	if _, err := toml.DecodeFile(planPath, &plan); err != nil {
		log.Fatal(err)
	}

	// Process each feature
	for _, feature := range featuresJson.Features {
		buildFeature(feature, &plan, layersDir, envDir, featuresPath)
	}

	// Write the updated build.toml
	file, err := os.OpenFile(filepath.Join(layersDir, "build.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(plan)
}

func buildFeature(feature FeatureConfig, plan *libcnb.BuildpackPlan, featuresPath string, layersDir string, envDir string) {
	createLayer, layer := createFeatureLayer(feature, plan)
	if createLayer {
		// Execute acquire script if present
		acquireScriptPath := filepath.Join(featuresPath, feature.Id, "bin", "acquire")
		_, err := os.Stat(acquireScriptPath)
		if err != nil {
			layer.Name = feature.Id
			env := os.Environ()
			env = append(env, "_BUILD_ARG_"+feature.Id+"=true")
			// TODO: Inspect devcontainer.json if present to find options
			// Current working directory is app directory
			syscall.Exec(acquireScriptPath, []string{}, env)
			// Write <layer-id>.toml
			file, err := os.OpenFile(filepath.Join(layersDir, "build.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				log.Fatal(err)
			}
			toml.NewEncoder(file).Encode(plan)
		}
	}
}

// See if the build plan includes an entry for this feature. If so, remove it
// from the plan and return an initalized layer for use in this buildpack
func createFeatureLayer(feature FeatureConfig, plan *libcnb.BuildpackPlan) (bool, libcnb.Layer) {
	var layer libcnb.Layer
	createLayer := false
	// See if detect said should provide this feature
	i := len(plan.Entries)
	for i > 0 {
		i--
		entry := plan.Entries[i]
		if entry.Name == feature.Id {
			plan.Entries = plan.Entries[:i]
			createLayer = true
			layer.LayerTypes.Build = entry.Metadata["Build"].(bool)
			layer.LayerTypes.Cache = entry.Metadata["Cache"].(bool)
			layer.LayerTypes.Launch = entry.Metadata["Launch"].(bool)
		}
	}

	return createLayer, layer
}
