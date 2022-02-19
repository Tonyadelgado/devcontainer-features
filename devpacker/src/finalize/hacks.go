package finalize

import (
	"encoding/json"
	"log"

	"github.com/chuxel/devpacker-features/devpacker/common"
)

/**
  Work around the fact that local file features are not yet implemented by mapping into devcontainer.json
  and there are properties in features.json missing from devcontainer.json
**/
func mergeFeatureConfigToDevContainerJson(postProcessingConfig PostProcessingConfig, devContainerJsonMap map[string]json.RawMessage) map[string]json.RawMessage {
	finalizeFeatureConfig := generateFinalizeFeatureConfig(postProcessingConfig)
	var runArgs []string
	if devContainerJsonMap["runArgs"] != nil {
		if err := json.Unmarshal(devContainerJsonMap["runArgs"], &runArgs); err != nil {
			log.Fatal("Failed to unmarshal runArgs from devcontainer.json: ", err)
		}
	}
	if finalizeFeatureConfig.Privileged {
		runArgs = common.AddToSliceIfUnique(runArgs, "--privileged")
	}
	if finalizeFeatureConfig.Init {
		runArgs = common.AddToSliceIfUnique(runArgs, "--init")
	}
	if finalizeFeatureConfig.CapAdd != nil {
		for _, cap := range finalizeFeatureConfig.CapAdd {
			runArgs = common.AddToSliceIfUnique(runArgs, "--cap-add="+cap)
		}
	}
	if finalizeFeatureConfig.SecurityOpt != nil {
		for _, opt := range finalizeFeatureConfig.SecurityOpt {
			runArgs = common.AddToSliceIfUnique(runArgs, "--security-opt="+opt)
		}
	}
	devContainerJsonMap["runArgs"] = common.ToJsonRawMessage(runArgs)

	if finalizeFeatureConfig.Extensions != nil {
		var extensions []string
		if devContainerJsonMap["extensions"] != nil {
			if err := json.Unmarshal(devContainerJsonMap["extensions"], &extensions); err != nil {
				log.Fatal("Failed to unmarshal extensions from devcontainer.json: ", err)
			}
		}
		extensions = common.SliceUnion(extensions, finalizeFeatureConfig.Extensions)
		devContainerJsonMap["extensions"] = common.ToJsonRawMessage(extensions)
	}
	if finalizeFeatureConfig.Settings != nil {
		var settings map[string]interface{}
		if devContainerJsonMap["settings"] != nil {
			if err := json.Unmarshal(devContainerJsonMap["settings"], &settings); err != nil {
				log.Fatal("Failed to unmarshal extensions from devcontainer.json: ", err)
			}
		}
		//TODO: Settings merge
		devContainerJsonMap["settings"] = common.ToJsonRawMessage(settings)
	}
	if finalizeFeatureConfig.Mounts != nil {
		var mounts []string
		if devContainerJsonMap["mounts"] != nil {
			if err := json.Unmarshal(devContainerJsonMap["mounts"], &mounts); err != nil {
				log.Fatal("Failed to unmarshal mounts from devcontainer.json: ", err)
			}
		}
		for _, mount := range finalizeFeatureConfig.Mounts {
			mounts = common.AddToSliceIfUnique(runArgs, "source="+mount.Source+",target="+mount.Target+",type="+mount.Type)
		}
		devContainerJsonMap["mounts"] = common.ToJsonRawMessage(mounts)
	}
	return devContainerJsonMap
}
