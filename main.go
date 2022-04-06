package main

import (
	"flag"
	"log"
	"os"

	licensecollector "github.com/aviadl/thirdPartyLicenseCollector/license-collector"
)

func main() {
	tmpGoDir := flag.String("go-project", "", "project directory")
	tmpNpmDir := flag.String("npm-project", "", "npm directory")
	out := flag.String("out", licensecollector.LicenseFileName, "output file")
	format := flag.String("format", licensecollector.DefaultLicenseFileFormat, "output format: text vs json")
	flag.Parse()
	log.SetFlags(0)

	err := licensecollector.Collect(*tmpGoDir, *tmpNpmDir, *out, *format)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
