# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.18.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.17.0...v0.18.0) (2026-07-24)


### Features

* acceptance tests ([#102](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/102)) ([9d79744](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9d797442cfe0f80486417409f364d94360df084f))
* emittance consolidation ([#100](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/100)) ([89ce179](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/89ce1792f88f03d9f920adbcf977e645b1454d35))
* emittance standardisation ([#94](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/94)) ([ac842cd](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/ac842cd036034b63939a295c581336634f51bab5))
* finish up ([#96](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/96)) ([e73fc58](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/e73fc587cb8ab643a1bb2778215d9ff0e2b9e205))
* **idiomatic:** CFTypeRef polymorphism + mutable-&gt;immutable handle widening ([#98](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/98)) ([a44eafa](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/a44eafad81cb1c17e9ba042333122f8ed1073b3e))
* **idiomatic:** surface CF/opaque handles as distinct named types, not obj.Object ([7e6fd7c](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/7e6fd7c76048b1779a4cdcfeb3c2f15353d1c7d6))
* **idiomatic:** surface dispatch/xpc handle params as concrete library types ([#91](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/91)) ([17a8326](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/17a832630c51df470bc02ae6eccdf5403dc8eef0))
* **idiomatic:** thread in/out counts, typed block params, pointer returns ([#88](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/88)) ([3c6138c](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/3c6138cb3364b9e72b0ecf528aa27e8112857fd9))
* **idiomatic:** type fixed-size array struct fields & struct-pointer params ([#92](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/92)) ([e56e3f3](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/e56e3f3f1990e04384be4199156c4956d79c528c))
* **idiomatic:** type pointer-to-struct returns; capture referenced C structs ([#90](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/90)) ([3045aed](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/3045aed922f956e2614141b6f9575ac7567c048b))
* **idiomatic:** width-correct struct int fields; cross-check clang layouts ([#93](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/93)) ([9379aad](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9379aad8ab474624e61f6196ec44f81d7846147c))
* unions fnptrs ([#99](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/99)) ([c4d649c](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/c4d649c1cfa39a89716bd98f193b210465ed8417))


### Bug Fixes

* **frameworks:** width-correct integer-typedef struct fields via GoABIType ([#95](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/95)) ([91d354a](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/91d354a5f0740cbfb52560d3b84cb4786368a0c7))

## [0.17.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.16.0...v0.17.0) (2026-07-21)


### Features

* make the idiomatic layer the sole public API under bindings/ ([#82](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/82)) ([2f52d47](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/2f52d47a8e55e0ba22e3e42eb47d25a1d63a04a6))
* **parity:** close all 276 residual gaps — idiomatic now covers raw fully ([82f6513](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/82f65134d691adeec1ed9a3a455310b369913f73))


### Bug Fixes

* **idiomatic:** restore typedef-alias gather gate to stop library over-emission ([#86](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/86)) ([9c28b69](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9c28b69f92e7d81dd82b1f5b4f7e875245635768))

## [0.16.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.15.0...v0.16.0) (2026-07-09)


### Features

* **idiomatic:** evolve emitted layer to hand-crafted-quality Go ([#77](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/77)) ([9391398](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/9391398cdc817b7b0e2fdafc7f70752b73d3c677))

## [0.15.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.14.0...v0.15.0) (2026-07-02)


### Features

* **bindings:** add mach, IOKit PowerSources, and IOReport telemetry surfaces ([#74](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/74)) ([2f16efc](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/2f16efc3799f2f1f02148b63dc3d4cbe66ce8ce4))

## [0.13.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.12.1...v0.13.0) (2026-06-29)


### Features

* **idiomatic:** auto-dispatch @MainActor calls onto the main thread ([#69](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/69)) ([72e4f9d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/72e4f9d90443e0e64910d43070565742796cc324))
* refactored code emitance ([#71](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/71)) ([fe3c6d2](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/fe3c6d23ddcf1db08bce8369b1bdfe60d9fad58b))

## [0.12.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.12.0...v0.12.1) (2026-06-25)


### Bug Fixes

* **codegen/externs:** emit ObjC class pointer externs as typed accessors ([#65](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/65)) ([8d83b9d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/8d83b9d17f61e9b7b3d44198d32727a559bb99c0))

## [0.12.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.11.3...v0.12.0) (2026-06-25)


### Features

* **scanner:** add HeaderOverride support for frameworks without conventional umbrella headers ([#63](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/63)) ([767783f](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/767783fcfe4858e3badac4d30bb14d71c0f47708))

## [0.11.3](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.11.2...v0.11.3) (2026-06-25)


### Bug Fixes

* **idiomatic-codegen:** The idiomatic codegen synthesizes an fmt/obj stringer String() (returning -description) for every framework-root class, and dropped any ([#61](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/61)) ([6f0d301](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/6f0d301f19a6a490826812af0c2930ce18b5aac1))

## [0.11.2](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.11.1...v0.11.2) (2026-06-24)


### Bug Fixes

* - mapper.go: new GoABIType(qt, goType) — walks the typedef chain and returns the C-faithful Go width (int32/uint32) for 32-bit C ints, leaving genuine 64-bit types (long, NSInteger, ssize_t) as int. ([#59](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/59)) ([597e684](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/597e68443b10c5cdda1e1375bcac70cad22e14d1))

## [0.11.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.11.0...v0.11.1) (2026-06-24)


### Bug Fixes

* **codegen/idiomatic:** pass (pointer, count) arrays as input, not out-params ([#57](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/57)) ([e4134fa](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/e4134fa42e467ea51d57021529e4ff31f012362e))

## [0.11.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.10.2...v0.11.0) (2026-06-24)


### Features

* **idiomatic:** update function signatures to use *bool for out-parameters ([#55](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/55)) ([4f618c2](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/4f618c2db420db2d50202de2fb67753bc75294c7))

## [0.10.2](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.10.1...v0.10.2) (2026-06-22)


### Bug Fixes

* **idiomatic:** synthesize doc comments and drop Get prefix from getters (Effective Go) ([#53](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/53)) ([6dec9ee](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/6dec9ee474aabce697fd68d6c957b09d88065bdb))

## [0.10.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.10.0...v0.10.1) (2026-06-22)


### Bug Fixes

* **idiomatic:** make emitted Go more idiomatic (Effective Go pass) ([#51](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/51)) ([40d8072](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/40d8072cd2e6dae3cc5bf02099255b13ff57e2f5))

## [0.10.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.9.0...v0.10.0) (2026-06-22)


### Features

* rebuild emitter as a view→render compiler ([#48](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/48)) ([d16253e](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/d16253ee7b57b4eb93bf8552d61660cda4294731))

## [0.9.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.8.2...v0.9.0) (2026-06-19)


### Features

* **mainthread:** add purego PumpMainRunLoop to opinionated/custom/mainthread ([#44](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/44)) ([4c5080f](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/4c5080f3b29e7d400a923a28ba3aaebdeb914ad8))

## [0.8.2](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.8.1...v0.8.2) (2026-06-19)


### Bug Fixes

* full regen of apple docs, frameworks and libraries ([#39](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/39)) ([f5aade3](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/f5aade3a2992302b2a50a7c57bebc37c1a0c1051))
* full regen of apple docs, frameworks and libraries ([#41](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/41)) ([5774b63](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/5774b6358e832d77836f22f0f74cd100fa84f410))
* tidy up stray binaries ([#42](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/42)) ([c404090](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/c40409083cfb58b15c9a1c6c19b4f0e556aad968))

## [0.8.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.8.0...v0.8.1) (2026-06-19)


### Bug Fixes

* **idiomatic:** re-export error-code enums ([#37](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/37)) ([196b5c2](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/196b5c29d3e440c2c67af778a8e7f4f721ffb6e0))

## [0.8.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.7.1...v0.8.0) (2026-06-19)


### Features

* **idiomatic:** emit struct aliases, cross-framework NSString externs, dict ergonomics, IDer dict ctor params ([#35](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/35)) ([55fa7f2](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/55fa7f26218a66a408dd1830fb6b9b6a099163e2))

## [0.7.1](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.7.0...v0.7.1) (2026-06-19)


### Bug Fixes

* **idiomatic:** emit empty NSArray (not nil) for empty collection setters ([#33](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/33)) ([c150517](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/c1505175fdc43c78976860288d3fa668e92d2d0a))

## [0.7.0](https://github.com/deploymenttheory/go-bindings-macosplatform/compare/v0.6.0...v0.7.0) (2026-06-18)


### Features

* Added SDK feature: idiomatic NSString extern constants ([#31](https://github.com/deploymenttheory/go-bindings-macosplatform/issues/31)) ([6eaae6d](https://github.com/deploymenttheory/go-bindings-macosplatform/commit/6eaae6d65c8f752df22e17352c80749d50469ef1))

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
