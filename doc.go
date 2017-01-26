// Proteus /proʊtiəs/ is a tool to generate protocol buffers version 3
// compatible `.proto` files from your Go structs, types and functions.
//
// The motivation behind this library is to use Go as a source of truth for
// your models instead of the other way around and then generating Go code
// from a `.proto` file, which does not generate idiomatic code.
//
// Proteus scans all the code in the selected packages and generates protobuf
// messages for every exported struct (and all the ones that are referenced in
// any other struct, even though they are not exported). The types that
// semantically are used as enumerations in Go are transformed into proper
// protobuf enumerations.
// All the exported functions and methods will be turned into protobuf RPC
// services.
package proteus
