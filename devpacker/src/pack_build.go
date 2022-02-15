package main

import (
	"log"
	"os"
	"os/exec"
)

func PackBuild(imageName string, buildMode string, applicationFolder string, packArgs []string) {
	log.Println("Image name:", imageName)
	log.Println("Application folder:", applicationFolder)
	log.Println("Pack CLI arguments:", packArgs)
	execPackBuild(imageName, buildMode, applicationFolder, packArgs)
	FinalizeImage(imageName, buildMode, applicationFolder)
}

func execPackBuild(imageName string, buildMode string, applicationFolder string, packArgs []string) {
	args := []string{"build", imageName}
	if buildMode != "" {
		packArgs = append(args, "-e", ContainerImageBuildModeEnvVarName+"="+buildMode)
	}
	args = append(args, packArgs...)
	// Invoke dev container CLI
	packCommand := exec.Command("pack", args...)
	packCommand.Env = os.Environ()
	writer := log.Writer()
	packCommand.Stdout = writer
	packCommand.Stderr = writer
	packCommand.Dir = applicationFolder
	commandErr := packCommand.Run()

	// Report command error if there was one
	if commandErr != nil || packCommand.ProcessState.ExitCode() != 0 {
		log.Fatal("Failed to build using pack CLI. " + commandErr.Error())
	}
}
