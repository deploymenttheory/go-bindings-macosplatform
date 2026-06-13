//go:build ignore

package main

import (
	"fmt"
	"os/exec"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/scanner"
)

func main() {
	sdk, _ := exec.Command("xcrun", "--show-sdk-path").Output()
	sdkPath := string(sdk[:len(sdk)-1])

	root, err := scanner.DumpAST(sdkPath, "Virtualization", "arm64")
	if err != nil {
		panic(err)
	}

	// Check whether inner is nil vs empty for known forward-decl and zero-method classes
	targets := map[string]bool{
		"NSScreen":        true,
		"VZXHCIController": true,
		"VZVirtioGraphicsDevice": true,
		"VZMacGraphicsDevice": true,
		"VZVirtualMachineStartOptions": true,
	}
	seen := map[string]bool{}
	for i := range root.Inner {
		n := &root.Inner[i]
		if n.Kind != "ObjCInterfaceDecl" || !targets[n.Name] {
			continue
		}
		key := fmt.Sprintf("%s@%d", n.Name, i)
		if seen[key] { continue }
		seen[key] = true

		loc := ""
		if n.Loc != nil { loc = n.Loc.FilePath }
		super := "<nil>"
		if n.Super != nil { super = n.Super.Name }
		innerNil := n.Inner == nil
		fmt.Printf("name=%-40s loc_short=%-40s super=%-15s inner_nil=%v inner_count=%d\n",
			n.Name, shortPath(loc), super, innerNil, len(n.Inner))
	}
}

func shortPath(p string) string {
	for _, prefix := range []string{
		"/Applications/Xcode.app/Contents/Developer/Platforms/MacOSX.platform/Developer/SDKs/MacOSX.sdk/System/Library/Frameworks/",
	} {
		if len(p) > len(prefix) && p[:len(prefix)] == prefix {
			return p[len(prefix):]
		}
	}
	return p
}
