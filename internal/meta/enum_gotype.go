package meta

import "strconv"

// signedMax maps a signed Go integer type name to its maximum value, and
// unsignedFor maps it to the unsigned counterpart of the same width.
var (
	signedMax = map[string]uint64{
		"int8":  1<<7 - 1,
		"int16": 1<<15 - 1,
		"int32": 1<<31 - 1,
		"int64": 1<<63 - 1,
		"int":   1<<63 - 1, // darwin arm64: int is 64-bit
	}
	unsignedFor = map[string]string{
		"int8":  "uint8",
		"int16": "uint16",
		"int32": "uint32",
		"int64": "uint64",
		"int":   "uint",
	}
)

// NormaliseEnumGoTypes rewrites the Go type of any enum whose member values
// do not fit the signed type Clang reported. Clang's AST records some
// unsigned C enums (e.g. AppleArchive's uint64_t AAFlags bitmask, where
// AA_FLAG_VERBOSITY_2 = 1<<63) with a signed underlying type; emitting the
// constants under that signed Go type overflows and fails to compile.
//
// An enum is promoted to the unsigned type of the same width when it has no
// negative member and at least one member exceeds the signed maximum. Enums
// whose largest member also exceeds the same-width unsigned maximum are
// widened to uint64.
func NormaliseEnumGoTypes(framework *FrameworkMeta) {
	for name, enum := range framework.Enums {
		max, ok := signedMax[enum.GoType]
		if !ok {
			continue
		}
		var maxValue uint64
		hasNegative := false
		for _, member := range enum.Members {
			if _, err := strconv.ParseInt(member.Value, 0, 64); err == nil {
				if member.Value != "" && member.Value[0] == '-' {
					hasNegative = true
					break
				}
			}
			v, err := strconv.ParseUint(member.Value, 0, 64)
			if err != nil {
				continue
			}
			if v > maxValue {
				maxValue = v
			}
		}
		if hasNegative || maxValue <= max {
			continue
		}
		promoted := unsignedFor[enum.GoType]
		if maxValue > 2*max+1 { // exceeds the same-width unsigned range
			promoted = "uint64"
		}
		enum.GoType = promoted
		framework.Enums[name] = enum
	}
}
