# NUMA

[![Build Status](https://travis-ci.org/lrita/numa.svg?branch=master)](https://travis-ci.org/lrita/numa)

NUMA is a utility library, which is written in go. It help us to write
some NUMA-AWARED code.

example gist:
```go
package main

import (
	"github.com/lrita/numa"
)

type object struct {
	X int
	_ [...]byte // padding to page size.
 }

var objects = make([]object, numa.CPUCount())

func fnxxxx() {
	cpu, node := numa.GetCPUAndNode()
	objects[cpu].X = xx
}
```
