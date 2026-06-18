# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.6.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.5.0...v0.6.0) (2026-06-18)


### Features

* enhance Apple developer documentation integration ([#27](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/27)) ([634a767](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/634a767369ee11c2a90d2b98c535289e74b838dc))

## [0.5.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.4.0...v0.5.0) (2026-06-18)


### Features

* Added Idiomatic CGo libraries ([fce2d31](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/fce2d31f21f5382bdbccc2f1f67e538f02810f62))
* canonicalize generated Go via gofmt; emit concrete idiomatic enums ([#22](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/22)) ([63b5d5e](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/63b5d5e4d7b6ecafce5819e6cdc13da9f1d16fb8))
* enhance idiomatic generation and library handling ([a7c5f66](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/a7c5f66e0e28f98df1d74daf6ef5fbf65b0805c7))
* localize enum types in idiomatic signatures ([#25](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/25)) ([c89b8c7](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/c89b8c753a1046db50c21ea318199fc811091afc))
* refactor(codegen): fully templatize the idiomatic emitter (method family, class wrappers, function wrappers) ([#24](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/24)) ([509bf9a](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/509bf9a4acf5e8a2bbf2c7f2f8e5025820c0b267))
* templatize idiomatic constants/functions/constructor/with-setter renderers ([#23](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/23)) ([0f7a78e](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/0f7a78e914c88a40754be751963cdb7896e33039))


### Bug Fixes

* remove OpenTelemetry dependency and update CGo bindings ([f705d9e](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/f705d9e1289cf6ff6e63ab88226049cd4df3e0c9))
* removed otel from macOS library bindings ([4a465fc](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/4a465fc750e6a0b31f9d45d5d6c53cceea27c038))

## [0.4.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.3.1...v0.4.0) (2026-06-17)


### Features

* Commit text (G1+G2 are the same theme — idiomatic types at the wrapper boundary — so one commit reads well): ([562ec5d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/562ec5dd9a23b02039fa5b5850dc09b7110ddd8e))
* return and accept idiomatic wrappers at slice/setter boundaries ([917de9c](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/917de9c2f564a65ff483329ed128863409463990))

## [0.3.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.3.0...v0.3.1) (2026-06-17)


### Bug Fixes

* flatten inherited setters onto subclass wrappers ([323e8dd](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/323e8dd0abe27929f54001c8954918f4416cb89b))
* flatten inherited setters onto subclass wrappers ([e98b593](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/e98b593c3970a72d4ed6416e4c67ab26ffb10a1b))

## [0.3.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.2.1...v0.3.0) (2026-06-17)


### Features

* added collections and delegates to idiomatic codegen ([c3cc892](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/c3cc89263d51925b1783d591b7a22dc6dd41bd6c))
* added collections and delegates to idiomatic codegen ([9235b43](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9235b436a8759ffd07be07e4e1f97b2b452bed90))
* added keychain example and refinement to code gen comments ([ad80c4d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/ad80c4df246523fa26702894b06233fd1d19b950))
* enhance keychain example with comprehensive CRUD operations across item classes ([3d253f6](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/3d253f696fce632dba941f7ff9a8e0413937aca2))
* regen ([9415eda](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9415eda6c536145ec1f38ded568e78f5aa04ac23))

## [0.2.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.2.0...v0.2.1) (2026-06-13)


### Bug Fixes

* added reexport of dynamic-loading from purego for consistency ([1b96a6d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/1b96a6d8a38e12bb24e630578baa0fa6218b7a91))
* added reexport of dynamic-loading from purego for consistency ([82e216d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/82e216da306c1eacb15ce3cfaf579c542ea3b448))

## [0.2.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.1.1...v0.2.0) (2026-06-13)


### Features

* Full summary of all changes this session ([7607ed2](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/7607ed2402a1314ce76643adc2f1797709885548))
* regen of code with tighter naming conventions ([9e28a13](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9e28a13dab6bede877412aa8c8f6b57597674239))

## [0.1.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.1.0...v0.1.1) (2026-06-13)


### Bug Fixes

* for release please and go mod ([97e6d7f](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/97e6d7f7f61c4f8aaae2ee89392aee75129e8d57))
* for release please and go mod ([4945824](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/49458248f6a8218c058d5ae018bb29ca60fa377c))

## [Unreleased]

### Added

- Added xyz [@your_username](https://github.com/your_username)

### Fixed

- Fixed zyx [@your_username](https://github.com/your_username)

## [1.1.0] - 2021-06-23

### Added

- Added x [@your_username](https://github.com/your_username)

### Changed

- Changed y [@your_username](https://github.com/your_username)

## [1.0.0] - 2021-06-20

### Added

- Inititated y [@your_username](https://github.com/your_username)
- Inititated z [@your_username](https://github.com/your_username)
