// Proteus /proʊtiəs/ is a tool to generate protocol buffers version 3 compatible `.proto` files from your Go structs and types.
//
// The motivation behind this library is to use Go as a source of truth for your models instead of the other way around and then generating Go code from a `.proto` file, which does not generate idiomatic code.
//
// Proteus scans all the code in the selected packages and generates protobuf messages for every exported struct (and all the ones that are referenced in any other struct, even though they are not exported). Also, the types that semantically are used as enumerations in Go are transformed into proper protobuf enumerations.
//
// We want to build proteus in a very extensible way, so every step of the generation can be hackable via plugins and everyone can adapt proteus to their needs without actually having to integrate functionality that does not play well with the core library. We are releasing the plugin feature after Go 1.8 is released, which includes the `plugin` package of the standard library.
//
// Install
//
//	go get -v github.com/src-d/proteus/...
//
// Usage
//
//	proteus -f /path/to/output/folder \
//        -p my/go/package \
//        -p my/other/go/package
//        --verbose
package proteus
