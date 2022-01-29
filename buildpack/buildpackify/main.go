package main

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/chuxel/buildpackify-features/libbuildpackify"

	"github.com/BurntSushi/toml"
	"github.com/buildpacks/libcnb"
)

const defaultApiVersion = "0.7"

func main() {
	featuresPath := "."
	if len(os.Args) > 0 {
		featuresPath = os.Args[1]
	}
	// Load features.json, buildpack settings
	featuresJson := libbuildpackify.LoadFeaturesJson(filepath.Join(featuresPath, "features.json"))
	buildpackSettings := libbuildpackify.LoadBuildpackSettings(filepath.Join(featuresPath, "buildpack-settings.json"))

	tmpDir := os.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755)
	for _, sourcePath := range []string{"devcontainer-features.json", "features.json", "buildpack-settings.json", "features", "common"} {
		cpR(filepath.Join(featuresPath, sourcePath), tmpDir)
	}

	var buildpack libcnb.Buildpack
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
		featureNameList = append(featureNameList, feature.Name)
	}
	buildpack.Metadata = make(map[string]interface{})
	buildpack.Metadata["com.microsoft.devcontainer.featureset"] = buildpackSettings
	buildpack.Metadata["com.microsoft.devcontainer.features"] = featureNameList

	file, err := os.OpenFile(filepath.Join(tmpDir, "buildpack.toml"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
		cp(sourcePath, filepath.Join(targetFolderPath, sourceFileInfo.Name()))
		return
	}

	// Handle if source is folder
	fileInfos, err := ioutil.ReadDir(sourcePath)
	if err != nil {
		log.Fatal(err)
	}
	for _, fileInfo := range fileInfos {
		fromPath := filepath.Join(sourcePath, fileInfo.Name())
		sourceFileInfo, err := os.Stat(fromPath)
		if err != nil {
			log.Fatal(err)
		}
		toPath := filepath.Join(targetFolderPath, fileInfo.Name())
		if sourceFileInfo.IsDir() {
			os.MkdirAll(toPath, sourceFileInfo.Mode())
			cpR(fromPath, toPath)
		} else {
			cp(fromPath, toPath)
		}
	}
}

func cp(fromPath string, toPath string) {
	targetFile, err := os.Create(toPath)
	if err != nil {
		log.Fatal(err)
	}

	// Sync source and target file mode and ownership
	sourceFileInfo, err := os.Stat(fromPath)
	if err != nil {
		log.Fatal(err)
	}
	targetFile.Chmod(sourceFileInfo.Mode())
	targetFile.Chown(int(sourceFileInfo.Sys().(*syscall.Stat_t).Uid), int(sourceFileInfo.Sys().(*syscall.Stat_t).Gid))

	// Execute copy
	sourceFile, err := os.Open(fromPath)
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(sourceFile, targetFile)
	if err != nil {
		log.Fatal(err)
	}
	targetFile.Close()
	sourceFile.Close()
}
