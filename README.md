# gonuma

[![License](http://img.shields.io/badge/license-mit-blue.svg)](https://raw.githubusercontent.com/johnsonjh/gonuma/master/LICENSE)
[![GoVersion](https://img.shields.io/github/go-mod/go-version/johnsonjh/gonuma.svg)](https://github.com/johnsonjh/gonuma/blob/master/go.mod)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/johnsonjh/gonuma)](https://pkg.go.dev/github.com/johnsonjh/gonuma)
[![GoReportCard](https://goreportcard.com/badge/github.com/johnsonjh/gonuma)](https://goreportcard.com/report/github.com/johnsonjh/gonuma)
[![LocCount](https://img.shields.io/tokei/lines/github/johnsonjh/gonuma.svg)](https://github.com/XAMPPRocky/tokei)
[![GitHubCodeSize](https://img.shields.io/github/languages/code-size/johnsonjh/gonuma.svg)](https://github.com/johnsonjh/gonuma)
[![CoverageStatus](https://coveralls.io/repos/github/johnsonjh/gonuma/badge.svg)](https://coveralls.io/github/johnsonjh/gonuma)
[![CodacyBadge](https://api.codacy.com/project/badge/Grade/6a688d07faaa4e848f59ec49fdb663bc)](https://app.codacy.com/gh/johnsonjh/gonuma?utm_source=github.com&utm_medium=referral&utm_content=johnsonjh/gonuma&utm_campaign=Badge_Grade)
[![CodeBeat](https://codebeat.co/badges/041414ca-af27-40f2-a5d6-13afc4ce9c6b)](https://codebeat.co/projects/github-com-johnsonjh-gonuma-master)
[![CodeclimateMaintainability](https://api.codeclimate.com/v1/badges/61db603e26c07e0e9ee4/maintainability)](https://codeclimate.com/github/johnsonjh/gonuma/maintainability)
[![TickgitTODOs](https://img.shields.io/endpoint?url=https://api.tickgit.com/badge?repo=github.com/johnsonjh/gonuma)](https://www.tickgit.com/browse?repo=github.com/johnsonjh/gonuma)

`gonuma` is a Go utility library for writing NUMA-aware applications

## Availability

* [Gridfinity GitLab](https://gitlab.gridfinity.com/jeff/go-numa)
* [GitHub](https://github.com/johnsonjh/gonuma)

## Original Author

* [lrita@163.com](https://github.com/lrita/numa)

## Usage

```go
 package main

 import (
    gonuma "github.com/johnsonjh/gonuma"
 )

 type object struct {
  X int
    _ [...]byte // pad to page size
  }

 var objects = make([]object, gonuma.CPUCount())

 func fnxxxx() {
    cpu, node := gonuma.GetCPUAndNode()
    objects[cpu].X = xx
 }
```
