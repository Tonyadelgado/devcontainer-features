package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/buildpacks/libcnb"
)

func main() {

	// Define flags
	var buildMode string
	flag.StringVar(&buildMode, "mode", "", "Override container image build mode: production | devcontainer")
	flag.Parse()
	nonFlagArgs := flag.Args()

	// First argument can be "generate", "finalize", "build", or "detect" - the latter two being internal only
	// No arguments is assumed to be "generate" to avoid confusion about the internal commands
	var commandName string
	if len(nonFlagArgs) < 1 {
		commandName = "generate"
	} else {
		commandName = nonFlagArgs[0]
	}

	switch commandName {
	case "generate":
		executeGenerateCommand(nonFlagArgs[1:])
	case "finalize":
		// Used to generate apply final build with the dev container CLI, output a devcontainer.json
		executeFinalizeCommand(nonFlagArgs[1:], buildMode)
	case "build":
		// Runs pack build and does any needed pre and post processing
		executePackBuildCommand(nonFlagArgs[1:], buildMode)
	case "_internal":
		// If doing a build or detect command, pass of processing to FeatureBuilder, FeatureDetector respectively
		libcnb.Main(FeatureDetector{}, FeatureBuilder{}, libcnb.WithArguments(nonFlagArgs[1:]))
	default:
		fmt.Println("Invalid devpacker command:", nonFlagArgs[0])
	}
}

func executeGenerateCommand(args []string) {
	featuresPath := "."
	outputPath := "out"
	if len(args) > 0 {
		featuresPath = args[0]
	}
	if len(args) > 1 {
		outputPath = args[1]
	}
	Generate(featuresPath, outputPath)
}

func executeFinalizeCommand(args []string, buildMode string) {
	applicationFolder := "."
	if len(args) < 1 {
		fmt.Println("Missing required parameter. Usage: devpacker finalize <image ID> [application folder]")
		os.Exit(1)
	}
	if len(args) > 1 {
		applicationFolder = args[1]
	}
	FinalizeImage(args[0], buildMode, applicationFolder)
}

func executePackBuildCommand(args []string, buildMode string) {
	if len(args) < 1 {
		fmt.Println("Missing required parameter. Usage: devpacker build <image ID>")
		os.Exit(1)
	}
	os.Setenv(ContainerImageBuildModeEnvVarName, buildMode)
	PackBuild(args[0], buildMode, args[1:])
}