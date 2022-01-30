package main

import (
	"os"

	"github.com/buildpacks/libcnb"
)

func main() {
	// First argument can be "generate", "build", or "detect" - the latter two being internal only
	// No arguments is assumed to be "generate" to avoid confusion about the internal commands
	if len(os.Args) > 1 && os.Args[1] != "generate" {
		// If doing a build or detect command, pass of processing to FeatureBuilder, FeatureDetector respectively
		os.Args = os.Args[1:]
		libcnb.Main(FeatureDetector{}, FeatureBuilder{})
		return
	}

	// Otherwise generate the buildpack
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
