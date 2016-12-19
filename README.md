# ![proteus](https://rawgit.com/src-d/proteus/master/proteus.svg) 

[![GoDoc](https://godoc.org/github.com/src-d/proteus?status.svg)](https://godoc.org/github.com/src-d/proteus) [![Build Status](https://travis-ci.org/src-d/proteus.svg?branch=master)](https://travis-ci.org/src-d/proteus) [![codecov](https://codecov.io/gh/src-d/proteus/branch/master/graph/badge.svg)](https://codecov.io/gh/src-d/proteus) [![License](http://img.shields.io/:license-mit-blue.svg)](http://doge.mit-license.org) [![Go Report Card](https://goreportcard.com/badge/github.com/src-d/proteus)](https://goreportcard.com/report/github.com/src-d/proteus) [![codebeat badge](https://codebeat.co/badges/976ff535-c79b-429d-b35c-888a048a3201)](https://codebeat.co/projects/github-com-src-d-proteus)

[Proteus](https://en.wikipedia.org/wiki/Proteus) /proʊtiəs/ is a tool to generate protocol buffers version 3 compatible `.proto` files from your Go structs and types.

### Usage

```bash
go get github.com/src-d/proteus/...

proteus -f /path/to/output/folder \
        -p my/go/package \
        -p my/other/go/package
```

### Features to come

- [ ] Extensible mapping and options via plugins (waiting for Go 1.8 release).
