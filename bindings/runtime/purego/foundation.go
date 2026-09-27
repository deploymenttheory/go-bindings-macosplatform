//go:build darwin

package purego

import (
	"sync"

	ebipurego "github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// loadFoundation loads Foundation, which also brings in CoreFoundation.
var loadFoundation = sync.OnceFunc(func() {
	_, _ = ebipurego.Dlopen("/System/Library/Frameworks/Foundation.framework/Foundation", ebipurego.RTLD_GLOBAL|ebipurego.RTLD_LAZY)
})

// FoundationClass returns the named Foundation or CoreFoundation class, such
// as NSString or NSURL, loading Foundation first. A process starts with only
// libobjc loaded, so looking such a class up before anything has loaded
// Foundation, for example in a package-level variable, yields nil. Generated
// framework packages follow the same rule for their own classes.
func FoundationClass(name string) Class {
	loadFoundation()
	return objc.GetClass(name)
}
