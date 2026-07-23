package meta

import (
	rootmeta "github.com/deploymenttheory/go-bindings-macosplatform/internal/macosplatformmetadata"
)

// The metadata model is owned by internal/macosplatformmetadata (the canonical
// copy the scanner writes and both pipelines read). This package used to keep a
// hand-maintained parallel copy of the same structs with identical JSON tags,
// which was a standing drift risk. It now re-exports the canonical types as
// aliases, so `meta.X` and `macosplatformmetadata.X` are the *same* type — the
// hundreds of existing `meta.*` references compile unchanged, and a framework
// value can be passed wherever a canonical value is expected (and vice versa).
//
// Read (io.go) and the schema-version window still live here so the frameworks
// pipeline has a single import surface for loading .gometa.json.
type (
	FrameworkMeta = rootmeta.FrameworkMeta
	Class         = rootmeta.Class
	Protocol      = rootmeta.Protocol
	Method        = rootmeta.Method
	Param         = rootmeta.Param
	ReturnType    = rootmeta.ReturnType
	Property      = rootmeta.Property
	Enum          = rootmeta.Enum
	EnumMember    = rootmeta.EnumMember
	Struct        = rootmeta.Struct
	StructField   = rootmeta.StructField
	Function      = rootmeta.Function
	Extern        = rootmeta.Extern
	BlockType     = rootmeta.BlockType
	Availability  = rootmeta.Availability
)
