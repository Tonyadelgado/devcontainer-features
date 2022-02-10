package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"

	_ "embed"
)

//go:embed assets/bin/detect.sh
var detectScriptPayload []byte

//go:embed assets/bin/build.sh
var buildScriptPayload []byte

func Generate(featuresPath string, outputPath string) {
	// Load features.json, buildpack settings
	featuresJson := LoadFeaturesJson(featuresPath)
	buildpackSettings := LoadBuildpackSettings(featuresPath)

	// Copy core features content
	os.MkdirAll(filepath.Join(outputPath, "bin"), 0755)
	for _, sourcePath := range []string{"devcontainer-features.json", "buildpack-settings.json", "features", "common"} {
		CpR(filepath.Join(featuresPath, sourcePath), outputPath)
	}
	// Output embedded bin/detect and bin/build files
	if err := WriteFile(filepath.Join(outputPath, "bin", "build"), buildScriptPayload); err != nil {
		log.Fatal(err)
	}
	if err := WriteFile(filepath.Join(outputPath, "bin", "detect"), detectScriptPayload); err != nil {
		log.Fatal(err)
	}

	// Copy all architecture versions of current executable, unless in debug mode where we should just copy this binary
	currentExecutableName := filepath.Base(os.Args[0])
	if strings.HasPrefix(currentExecutableName, "buildpackify-linux-") {
		currentExecutablePath := filepath.Dir(os.Args[0])
		fileInfos, err := ioutil.ReadDir(currentExecutablePath)
		if err != nil {
			log.Fatal(err)
		}
		for _, fileInfo := range fileInfos {
			if strings.HasPrefix(fileInfo.Name(), "buildpackify-linux-") {
				Cp(filepath.Join(currentExecutablePath, fileInfo.Name()), filepath.Join(outputPath, "bin"))
			}
		}
	} else {
		// This would typically happen when you are debugging where the file name will be different
		Cp(os.Args[0], filepath.Join(outputPath, "bin"))
		if err := os.Rename(filepath.Join(outputPath, "bin", currentExecutableName), filepath.Join(outputPath, "bin", "buildpackify-linux-"+runtime.GOARCH)); err != nil {
			log.Fatal(err)
		}
	}

	var buildpack libcnb.Buildpack
	buildpack.Info = libcnb.BuildpackInfo{
		ID:      buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet,
		Version: buildpackSettings.Version,
	}
	if buildpackSettings.ApiVersion != "" {
		buildpack.API = buildpackSettings.ApiVersion
	} else {
		buildpack.API = DefaultApiVersion
	}

	buildpack.Stacks = make([]libcnb.BuildpackStack, 0)
	for _, stack := range buildpackSettings.Stacks {
		buildpack.Stacks = append(buildpack.Stacks, libcnb.BuildpackStack{ID: stack})
	}
	var featureNameList []string
	for _, feature := range featuresJson.Features {
		featureNameList = append(featureNameList, feature.Id)
	}
	buildpack.Metadata = make(map[string]interface{})
	buildpack.Metadata[FeaturesetMetadataId] = buildpackSettings
	buildpack.Metadata[FeaturesMetadataId] = featureNameList

	// Write buildpack.toml - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpacktoml-toml
	file, err := os.OpenFile(filepath.Join(outputPath, "buildpack.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(buildpack)
}
