package main

const DefaultApiVersion = "0.7"

const MetadataIdPrefix = "com.microsoft.devcontainer"
const FeaturesetMetadataId = MetadataIdPrefix + ".featureset"
const FeaturesMetadataId = MetadataIdPrefix + ".features"
const FeatureLayerMetadataId = MetadataIdPrefix + ".feature"
const BuildModeMetadataId = MetadataIdPrefix + ".buildmode"
const PostProcessingDoneMetadataId = FeaturesMetadataId + ".done"

const OptionMetadataKeyPrefix = "option_"
const BuildpackDirEnvVar = "CNB_BUILDPACK_DIR"
const ContainerImageBuildModeEnvVarName = "BP_DCNB_BUILD_MODE"
const RemoveApplicationFolderOverrideEnvVarName = "BP_DCNB_OMIT_APP_DIR"
const OptionSelectionEnvVarPrefix = "_BUILD_ARG_"
const ProjectTomlOptionSelectionEnvVarPrefix = "BP_CONTAINER_FEATURE_"
const DefaultContainerImageBuildMode = "production"
const DevContainerConfigSubfolder = "/etc/dev-container-features"
const ContainerImageBuildMarkerPath = "/usr/local/" + DevContainerConfigSubfolder + "/dcnb-build-mode"
const DevpackSettingsFilename = "devpack-settings.json"
const BuildModeDevContainerJsonSetting = "buildMode"
const TargetPathDevContainerJsonSetting = "targetPath"
