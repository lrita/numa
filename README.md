# gonuma

[![License](http://img.shields.io/badge/license-mit-blue.svg)](https://raw.githubusercontent.com/johnsonjh/gonuma/master/LICENSE)
[![GoReportCard](https://goreportcard.com/badge/github.com/johnsonjh/gonuma)](https://goreportcard.com/report/github.com/johnsonjh/gonuma)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/6a688d07faaa4e848f59ec49fdb663bc)](https://app.codacy.com/gh/johnsonjh/gonuma?utm_source=github.com&utm_medium=referral&utm_content=johnsonjh/gonuma&utm_campaign=Badge_Grade)
[![CodeBeat](https://codebeat.co/badges/041414ca-af27-40f2-a5d6-13afc4ce9c6b)](https://codebeat.co/projects/github-com-johnsonjh-gonuma-master)
[![TickgitTODOs](https://img.shields.io/endpoint?url=https://api.tickgit.com/badge?repo=github.com/johnsonjh/gonuma)](https://www.tickgit.com/browse?repo=github.com/johnsonjh/gonuma)

`gonuma` is a Go utility library for writing NUMA-aware applications

## Availability

*  [Gridfinity GitLab](https://gitlab.gridfinity.com/jeff/go-numa)
*  [GitHub](https://github.com/johnsonjh/gonuma)

## Original Author

*  [lrita@163.com](https://github.com/lrita/numa)

## Usage

```go
 package main

 import (
    gonuma "github.com/johnsonjh/gonuma"
 )

 type object struct {
  X int
    _ [...]byte // padding to page size.
  }

 var objects = make([]object, gonuma.CPUCount())

 func fnxxxx() {
    cpu, node := gonuma.GetCPUAndNode()
    objects[cpu].X = xx
 }
```
