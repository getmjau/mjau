package main

import (
	"github.com/fr3h4g/mjau/cmd/mjau"
)

var Version string

func main() {
	mjau.Version = Version
	mjau.Execute()
}
