module github.com/chuxel/buildpackify-features/build

require (
	github.com/BurntSushi/toml v1.0.0
	github.com/buildpacks/libcnb v1.25.4
	github.com/chuxel/buildpackify-features/libbuildpackify v0.0.0
	github.com/onsi/gomega v1.17.0 // indirect
)

replace github.com/chuxel/buildpackify-features/libbuildpackify => ../../libbuildpackify

go 1.17