package main

import (
	"flag"
	"log"
	"os"

	licensecollector "github.com/demisto/thirdPartyLicenseCollector/license-collector"
)

func main() {
	tmpGoDir := flag.String("go-project", "", "project directory")
	tmpNpmDir := flag.String("npm-project", "", "npm directory")
	out := flag.String("out", licensecollector.LicenseFileName, "output file")
	flag.Parse()
	log.SetFlags(0)

	err := licensecollector.Collect(*tmpGoDir, *tmpNpmDir, *out)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
