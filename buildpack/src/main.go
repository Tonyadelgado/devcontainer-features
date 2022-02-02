package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/buildpacks/libcnb"
)

func main() {
	// First argument can be "generate", "finalize", "build", or "detect" - the latter two being internal only
	// No arguments is assumed to be "generate" to avoid confusion about the internal commands
	var commandName string
	if len(os.Args) < 2 {
		commandName = "generate"
	} else {
		commandName = os.Args[1]
	}

	switch commandName {
	case "generate":
		executeGenerateCommand()
		break
	case "finalize":
		// Used to generate apply final build with the dev container CLI, output a devcontainer.json
		executeFinalizeCommand()
		break
	default:
		// If doing a build or detect command, pass of processing to FeatureBuilder, FeatureDetector respectively
		buildpackArguments := os.Args[1:]
		libcnb.Main(FeatureDetector{}, FeatureBuilder{}, libcnb.WithArguments(buildpackArguments))
	}
}

func executeGenerateCommand() {
	featuresPath := "."
	outputPath := "out"
	if len(os.Args) > 2 {
		featuresPath = os.Args[2]
	}
	if len(os.Args) > 3 {
		outputPath = os.Args[3]
	}
	Generate(featuresPath, outputPath)
}

func executeFinalizeCommand() {
	// Define flags
	var buildMode string
	flag.StringVar(&buildMode, "mode", GetContainerImageBuildMode(), "Container image build mode: production | devcontainer")
	flag.Parse()

	applicationFolder := "."
	if len(os.Args) < 3 {
		fmt.Println("Missing required parameter. Usage: buildpackify finalize <image ID> [application folder]")
		os.Exit(1)
	}
	if len(os.Args) > 3 {
		applicationFolder = os.Args[3]
	}
	FinalizeImage(os.Args[2], applicationFolder, buildMode)
}
