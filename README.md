# gonuma

[![GoReportCard](https://goreportcard.com/badge/github.com/johnsonjh/gonuma)](https://goreportcard.com/report/github.com/johnsonjh/gonuma)

`gonuma` is a Go utility library for writing NUMA-aware applications


## Availability
  *  [Gridfinity GitLab](https://gitlab.gridfinity.com/jeff/go-numa)
  *  [GitHub](https://github.com/johnsonjh/gonuma)


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
		_ [...]byte // padding to page size.
	 }

	var objects = make([]object, gonuma.CPUCount())

	func fnxxxx() {
		cpu, node := gonuma.GetCPUAndNode()
		objects[cpu].X = xx
	}
```

