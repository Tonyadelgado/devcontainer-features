package common

// Defaults
const DefaultApiVersion = "0.7"
const DefaultContainerImageBuildMode = "production"

// Label and metadata keys
const MetadataIdPrefix = "com.microsoft.devcontainer"
const FeaturesetMetadataId = MetadataIdPrefix + ".featureset"
const FeaturesMetadataId = MetadataIdPrefix + ".features"
const FeatureLayerMetadataId = MetadataIdPrefix + ".feature"
const BuildModeMetadataId = MetadataIdPrefix + ".buildmode"
const PostProcessingDoneMetadataId = FeaturesMetadataId + ".done"

// ENV variables
const BuildpackDirEnvVar = "CNB_BUILDPACK_DIR"
const ContainerImageBuildModeEnvVarName = "BP_DCNB_BUILD_MODE"
const RemoveApplicationFolderOverrideEnvVarName = "BP_DCNB_OMIT_APP_DIR"
const OptionSelectionEnvVarPrefix = "_BUILD_ARG_"
const ProjectTomlOptionSelectionEnvVarPrefix = "BP_CONTAINER_FEATURE_"

// Property names
const BuildModeDevContainerJsonSetting = "buildMode"
const TargetPathDevContainerJsonSetting = "targetPath"

// TOML keys
const OptionMetadataKeyPrefix = "option_"

// Paths and filenames
const DevpackSettingsFilename = "devpack-settings.json"
const DevContainerConfigRelativeRoot = "/etc/dev-container-features"
const DevContainerFeatureConfigSubfolder = DevContainerConfigRelativeRoot + "/feature-config"
const ContainerImageBuildMarkerPath = "/usr/local" + DevContainerConfigRelativeRoot + "/dcnb-build-mode"
const DevContainerEntrypointD = DevContainerConfigRelativeRoot + "/entrypoint.d"
const CommonEntrypointDBootstrapPath = "/usr/local" + DevContainerConfigRelativeRoot + "/entrypoint-bootstrap.sh"
