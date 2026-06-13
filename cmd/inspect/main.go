// inspect prints a summary of a .gometa.json file.
//
// Usage:
//
//	go run ./cmd/inspect/ path/to/framework.gometa.json [--class ClassName]
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"github.com/deploymenttheory/go-bindings-macosplatform/internal/meta"
)

func main() {
	class := flag.String("class", "", "show details for a specific class")
	flag.Parse()

	path := flag.Arg(0)
	if path == "" {
		fmt.Fprintln(os.Stderr, "usage: inspect <path.gometa.json> [--class ClassName]")
		os.Exit(1)
	}

	framework, err := meta.Read(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read: %v\n", err)
		os.Exit(1)
	}

	if *class != "" {
		printClass(framework, *class)
		return
	}

	printSummary(framework)
}

func printSummary(framework *meta.FrameworkMeta) {
	fmt.Printf("Framework:  %s\n", framework.Framework)
	fmt.Printf("SDK:        %s\n", framework.SDKVersion)
	fmt.Printf("Arch:       %s\n", framework.Arch)
	fmt.Printf("Classes:    %d\n", len(framework.Classes))
	fmt.Printf("Protocols:  %d\n", len(framework.Protocols))
	fmt.Printf("Enums:      %d\n", len(framework.Enums))
	fmt.Printf("Structs:    %d\n", len(framework.Structs))
	fmt.Printf("Externs:    %d\n", len(framework.Externs))
	fmt.Printf("Functions:  %d\n", len(framework.Functions))
	fmt.Printf("BlockTypes: %d\n", len(framework.BlockTypes))
	fmt.Printf("Typedefs:   %d\n", len(framework.Typedefs))
	fmt.Println()

	names := make([]string, 0, len(framework.Classes))
	for k := range framework.Classes {
		names = append(names, k)
	}
	sort.Strings(names)
	fmt.Println("Classes:")
	for _, name := range names {
		cls := framework.Classes[name]
		generic := ""
		if len(cls.GenericParams) > 0 {
			generic = fmt.Sprintf("[%v]", cls.GenericParams)
		}
		super := cls.Super
		if super == "" {
			super = "(no super)"
		}
		fmt.Printf("  %-45s super=%-25s methods=%d%s\n",
			name+generic, super, len(cls.Methods), "")
	}
}

func printClass(framework *meta.FrameworkMeta, name string) {
	cls, ok := framework.Classes[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "class %q not found in %s metadata\n", name, framework.Framework)
		os.Exit(1)
	}

	fmt.Printf("Class:    %s\n", name)
	fmt.Printf("Super:    %s\n", cls.Super)
	if len(cls.GenericParams) > 0 {
		fmt.Printf("Generics: %v\n", cls.GenericParams)
	}
	if len(cls.Protocols) > 0 {
		fmt.Printf("Protocols: %v\n", cls.Protocols)
	}
	av := cls.Availability
	if av.MacOSIntroduced != "" {
		fmt.Printf("Introduced: macOS %s\n", av.MacOSIntroduced)
	}
	if av.MacOSDeprecated != "" {
		fmt.Printf("Deprecated: macOS %s\n", av.MacOSDeprecated)
	}
	fmt.Printf("Methods:  %d\n", len(cls.Methods))
	fmt.Println()

	for _, m := range cls.Methods {
		kind := "instance"
		if m.IsClassMethod {
			kind = "class"
		}
		extra := ""
		if m.IsNSError {
			extra += " [NSError]"
		}
		if m.IsVariadic {
			extra += " [variadic]"
		}
		fmt.Printf("  [%s] %s%s\n", kind, m.Selector, extra)
		fmt.Printf("          return: %s\n", m.Return.ObjCType)
		for _, arg := range m.Params {
			fmt.Printf("          arg %s: %s\n", arg.Name, arg.ObjCType)
		}
	}
}
