package main

import (
	"github.com/getmjau/mjau/cmd/mjau"
)

var Version string = "development"

func main() {
	mjau.Version = Version
	mjau.Execute()
}
