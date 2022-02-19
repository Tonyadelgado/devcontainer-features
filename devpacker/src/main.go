package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/buildpacks/libcnb"
	"github.com/chuxel/devpacker-features/devpacker/finalize"
	"github.com/chuxel/devpacker-features/devpacker/internal"
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
		libcnb.Main(internal.FeatureDetector{}, internal.FeatureBuilder{}, libcnb.WithArguments(nonFlagArgs[1:]))
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

func executeFinalizeCommand(args []string, buildModeOverride string) {
	var applicationFolder string
	if len(args) < 1 {
		fmt.Println("Missing required parameter. Usage: devpacker finalize <image ID> [application folder]")
		os.Exit(1)
	}
	if len(args) > 1 {
		applicationFolder = args[1]
	} else {
		if cwd, err := os.Getwd(); err != nil {
			log.Fatal("Unable got get current working directory.", err)
		} else {
			applicationFolder = cwd
		}
	}
	imageToFinalize := args[0]
	finalize.FinalizeImage(imageToFinalize, buildModeOverride, applicationFolder)
}

func executePackBuildCommand(args []string, buildModeOverride string) {
	var applicationFolder string
	var imageName string
	var flagArgs []string
	if len(args) < 1 {
		fmt.Println("Missing required parameter. Usage: devpacker build <image ID> [pack CLI args]")
		os.Exit(1)
	}
	for len(args) > 0 {
		if strings.HasPrefix(args[0], "-") {
			// Handle flags that have no value
			if len(args) == 1 || strings.HasPrefix(args[1], "-") {
				flagArgs = append(flagArgs, args[0])
				args = args[1:]
			} else {
				// Update application folder if -p or --path flag found
				if args[0] == "-p" || args[0] == "--path" {
					applicationFolder = args[1]
				}
				flagArgs = append(flagArgs, args[0], args[1])
				args = args[2:]
			}
		} else {
			imageName = args[0]
			args = args[1:]
		}
	}
	if applicationFolder == "" {
		if cwd, err := os.Getwd(); err != nil {
			log.Fatal("Unable got get current working directory.", err)
		} else {
			applicationFolder = cwd
		}
	}
	PackBuild(imageName, buildModeOverride, applicationFolder, flagArgs)
}
