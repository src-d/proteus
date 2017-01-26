# Proteus examples

This example shows how to use proteus for a typical use case. Imagine you have your Go models and you need to generate some `.proto` files to match that models.
It has some entities like the ones you could have on a database.

To generate the `.proto` you can use the `proteus` binary:

```
proteus proto -p github.com/src-d/proteus/example \
              -f $GOPATH/src/github.com/src-d/proteus/example/protos \
              --verbose
```

The generated file will be in:

```
$GOPATH/src/github.com/src-d/proteus/example/protos/github.com/proteus/example/generated.proto
```
