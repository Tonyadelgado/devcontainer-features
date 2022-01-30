package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"

	_ "embed"
)

const defaultApiVersion = "0.7"
const idPrefix = "com.microsoft.devcontainer"
const featuresetMetadataId = idPrefix + ".featureset"
const featuresMetadataId = idPrefix + ".features"

//go:embed assets/bin/build
var buildScriptPayload []byte

//go:embed assets/bin/detect
var detectScriptPayload []byte

func Generate(featuresPath string, outputPath string) {
	// Load features.json, buildpack settings
	featuresJson := LoadFeaturesJson(featuresPath)
	buildpackSettings := LoadBuildpackSettings(featuresPath)

	// Copy core features content
	os.MkdirAll(filepath.Join(outputPath, "bin"), 0755)
	for _, sourcePath := range []string{"devcontainer-features.json", "buildpack-settings.json", "features", "common"} {
		cpR(filepath.Join(featuresPath, sourcePath), outputPath)
	}
	// Output embedded bin/detect and bin/build files
	if err := writeFile(filepath.Join(outputPath, "bin", "build"), buildScriptPayload); err != nil {
		log.Fatal(err)
	}
	if err := writeFile(filepath.Join(outputPath, "bin", "detect"), detectScriptPayload); err != nil {
		log.Fatal(err)
	}

	// TODO: Copy binaries from dist folder

	var buildpack libcnb.Buildpack
	buildpack.Info = libcnb.BuildpackInfo{
		ID:      buildpackSettings.Publisher + "/" + buildpackSettings.FeatureSet,
		Version: buildpackSettings.Version,
	}
	if buildpackSettings.ApiVersion != "" {
		buildpack.API = buildpackSettings.ApiVersion
	} else {
		buildpack.API = defaultApiVersion
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
	buildpack.Metadata[featuresetMetadataId] = buildpackSettings
	buildpack.Metadata[featuresMetadataId] = featureNameList

	// Write buildpack.toml - https://github.com/buildpacks/spec/blob/main/buildpack.md#buildpacktoml-toml
	file, err := os.OpenFile(filepath.Join(outputPath, "buildpack.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}
	toml.NewEncoder(file).Encode(buildpack)
}

func cpR(sourcePath string, targetFolderPath string) {
	sourceFileInfo, err := os.Stat(sourcePath)
	if err != nil {
		// Return if source path doesn't exist so we can use this with optional files
		return
	}
	// Handle if source is file
	if !sourceFileInfo.IsDir() {
		cp(sourcePath, targetFolderPath)
		return
	}

	// Otherwise create the directory and scan contents
	toFolderPath := filepath.Join(targetFolderPath, sourceFileInfo.Name())
	os.MkdirAll(toFolderPath, sourceFileInfo.Mode())
	fileInfos, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(sourcePath, fileInfo.Name())
		if fileInfo.IsDir() {
			cpR(fromPath, toFolderPath)
		} else {
			cp(fromPath, toFolderPath)
		}
	}
}

func cp(sourceFilePath string, targetFolderPath string) {
	sourceFileInfo, err := os.Stat(sourceFilePath)
	if err != nil {
		log.Fatal(err)
	}

	// Make target folder
	targetFilePath := filepath.Join(targetFolderPath, sourceFileInfo.Name())
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		log.Fatal(err)
	}
	// Sync source and target file mode and ownership
	targetFile.Chmod(sourceFileInfo.Mode())
	targetFile.Chown(int(sourceFileInfo.Sys().(*syscall.Stat_t).Uid), int(sourceFileInfo.Sys().(*syscall.Stat_t).Gid))

	// Execute copy
	sourceFile, err := os.Open(sourceFilePath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(targetFile, sourceFile)
	if err != nil {
		log.Fatal(err)
	}
	targetFile.Close()
	sourceFile.Close()
}

func writeFile(filename string, bytes []byte) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err = file.Write(buildScriptPayload); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return nil
}
