# Proteus architecture

## proto generation

Generating a `.proto` file consists of four sequential steps.

```
+---------+
| scanner |
+---------+
     |
     v
+----------+
| resolver |
+----------+
     |
     v
+----------------------+
| protobuf transformer |
+----------------------+
     |
     v
+--------------------+
| protobuf generator |
+--------------------+
```

### `scanner`

First one is the `scanner`. The scanner reads the code in the given packages and extracts 3 things:

- `Struct`: all structs and their fields in the package. All of them, no matter if they have the `proteus:generate` comment or not. The ones that do have the `proteus:generate` are marked for generation directly in this step, though.
- Aliases: all named types that are _aliases_ of other types in the package (e.g. `type IntList []int`).
- `Enum`: all opted-in aliased types (e.g. `type A B`) with constant values in the package into aliases.
- `Func`: all functions and methods in the package.

What `scanner` builds is **not** a Go source representation. It's a representation of the entities we extract from Go source code.

All type types in Structs, Aliases and Functions are one of the following kinds:
- `Basic`: basic types of Go (e.g. `string`, `int`, ...).
- `Map`: key-value map between two types.
- `Named`: a type that has a name and references another type. Structs are always a `Named` type. For example, consider `type IntList []int`. In this step, `IntList` is a `Named`, even though after the `resolver` step it will be converted into a `Basic` repeated `int`.

All types can be repeated, which means they represent a repetition of values of the type.

In this step, packages are scanned in isolation, which is why they are scanned concurrently. Each individual package scan performs the scan of all the package and extracts values (`const` declarations), type aliases (`type A B`), structs, functions and methods. 
When all entities are scanned the enumerations are built, because you need to have all the collected entities first. When an enumeration is added, the alias corresponding to that type is removed, so all types with a `Named` referencing the enum are not converted to their `Basic` underlying type in the resolving step afterwards.

### `resolver`

Resolver is the second step in the process. It takes all packages that will be generated and resolves them all.
The `resolver` does 3 things:
- Changes all aliased types to their underlying type (e.g. in the case of `type IntList []int` it converts all the `Named` types referencing `IntList` to a repeated `Basic` of type `int`).
- Marks for generation every struct that does not have `proteus:generate` comment but is referenced in another that does have it.
- Ignores types that are not basic types, have been scanned and are not one of the custom types. For example, if you use the type `os.File` but `os` is not one of the scanned packages it can't be allowed further than this step.

Once all the packages are resolved they are marked as resolved and all the structs not marked for generation are removed.

**Custom types**

Custom types are types that may or may not be on the list of scanned packages but are always allowed.

- `time.Time`
- `time.Duration`
- `error`

In the future, the list will be extensible via plugins.

### `protobuf transformer`

`Transformer` is inside the `protobuf` package. Even though the step is transforming, it's inside the `protobuf` package to note that it's transforming the `scanner.Package` into a protobuf package. This opens the possibility of swapping the protobuf transformer for another, in the future.

What transformer does is convert from scanner types into protobuf representations that will later be used to generate a `.proto` file.

- `scanner.Struct` is converted to `protobuf.Message`.
- `scanner.Enum` is converted to `protobuf.Enum`.
- `scanner.Func` is converted to `protobuf.RPC`.

All types are also converted to protobuf types.

- `scanner.Basic` is converted to `protobuf.Basic`, which is now the protobuffer type name, instead of the Go type.
- `scanner.Named` is converted to `protobuf.Named`.
- `scanner.Map` is converted to `protobuf.Map`.

One important thing to mention is that `protobuf` types are **not repeated** even though their scanned type was. The `Field` of the `Message` is the one that knows whether the type of the field is repeated or not.

In the case of `protobuf.RPC`, as protobuf does not allow maps or basic types as input parameters or output parameters and only allows one single argument and one single return value, the `transformer` also adds additional `protobuf.Message`s for these.
For example, a function with the signature `func A(a int, b float64) (int, int)` would require to generate a message `ARequest` and `AResponse`.

### `protobuf generator`

`Generator` is also in the `protobuf` package for the same reasons `Transformer` is.

What Generator does is create the `.proto` file with the contents of the protobuf package representation.
**WARNING:** Generator has the side effect of actually writing the file.

## gRPC server implementation

Generating the gRPC server implementation consists of four sequential steps.

```
+---------+
| scanner |
+---------+
     |
     v
+----------+
| resolver |
+----------+
     |
     v
+----------------------+
| protobuf transformer |
+----------------------+
     |
     v
+---------------+
| rpc generator |
+---------------+
```

`scanner`, `resolver` and `protobuf transformer` are described in the previous section.

### `rpc generator`

`Generator` creates and writes the file with the Go RPC server implementation to disk.

The following things are implemented:

- `{serviceName}Server` struct with the first name in lowercase (e.g. `fooServiceServer` for a package named `foo`). This will only be implemented if there is no `{serviceName}Server` already implemented in the package.
- `New{ServiceName}Server` constructor returning `{serviceName}Server` with the first name of the service name in uppercase (e.g. `NewFooServiceServer` for a package named `foo`). This will only be implemented if there is no function named `New{ServiceName}Server` already implemented in the package.
- A method of `{serviceName}Server` for every generated function or method in the package.

When everything is generated, the file `server.proteus.go` is written in the corresponding package with the RPC server implementation.
