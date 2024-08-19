package main

import (
	"github.com/fr3h4g/mjau/cmd/mjau"
)

var Version string = "development"

func main() {
	mjau.Version = Version
	mjau.Execute()
}
