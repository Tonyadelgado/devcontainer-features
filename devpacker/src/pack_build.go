package main

import (
	"log"
)

func PackBuild(imageName string, buildMode string, args []string) {
	packArgs := []string{"build", imageName}
	if buildMode != "" {
		packArgs = append(packArgs, "-e", ContainerImageBuildModeEnvVarName+"="+buildMode)
	}

	// TODO: Implement
	log.Fatal("Not yet implmented.")

	FinalizeImage(imageName, buildMode, ".")
}
