package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanuber/go-license"
)

var exitOnMissing bool

func main() {
	tmpGoDir := flag.String("go-project", "", "project directory")
	tmpNpmDir := flag.String("npm-project", "", "npm directory")
	out := flag.String("out", "THIRD_PARTY_LICENSE", "output file")
	flag.Parse()
	log.SetFlags(0)
	licenseMap := map[string][]string{}
	foundManualLicense := map[string]string{}

	if len(*tmpGoDir) > 0 {
		CollectGoLicenseFiles(*tmpGoDir, licenseMap, foundManualLicense)
	}
	if len(*tmpNpmDir) > 0 {
		CollectNpmLicenseFiles(*tmpNpmDir, licenseMap, foundManualLicense)
	}
	if len(licenseMap)+len(foundManualLicense) > 0 {
		ioutil.WriteFile(*out, []byte(generateLicenseFile(licenseMap, foundManualLicense)), 0644)
	} else {
		log.Println("No licenses handled")
		exitOnMissing = true
	}
	if exitOnMissing {
		os.Exit(1)
	}
}

//CollectGoLicenseFiles ...
func CollectGoLicenseFiles(tmpGoDir string, licenseMap map[string][]string, foundManualLicense map[string]string) error {
	log.Println("Go Project dir: ", tmpGoDir)
	dir := filepath.Join(tmpGoDir, "vendor")
	fileName := filepath.Join(dir, "vendor.json")
	log.Println("Processing vendor file: ", fileName)
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println(err)
		log.Println("Failed processing go licenses")
		return err
	}

	vendorMap := map[string]interface{}{}
	err = json.Unmarshal(data, &vendorMap)
	//Get the package list
	rawPackages := vendorMap["package"]
	packages := rawPackages.([]interface{})

	manualLicense := prepareManualLicense(dir)
	for i := range packages {
		p := packages[i].(map[string]interface{})
		fileDir := p["path"].(string)
		doParseFile(dir, fileDir, manualLicense, licenseMap, foundManualLicense)
	}
	return nil
}

//CollectNpmLicenseFiles ...
func CollectNpmLicenseFiles(tmpNpmDir string, licenseMap map[string][]string, foundManualLicense map[string]string) error {
	log.Println("NPM Project dir: ", tmpNpmDir)
	dir := filepath.Join(tmpNpmDir, "node_modules")
	fileName := filepath.Join(tmpNpmDir, "package.json")
	log.Println("Processing package file: ", fileName)
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println(err)
		log.Println("Failed processing npm licenses")
		return err
	}

	packageMap := map[string]interface{}{}
	err = json.Unmarshal(data, &packageMap)
	//Get the package list
	rawPackages := packageMap["dependencies"]
	packages := rawPackages.(map[string]interface{})

	manualLicense := prepareManualLicense(tmpNpmDir)
	for fileDir := range packages {
		doParseFile(dir, fileDir, manualLicense, licenseMap, foundManualLicense)
	}
	return nil
}

func doParseFile(dir, fileDir string, manualLicense map[string]string, licenseMap map[string][]string, foundManualLicense map[string]string) {
	lDir, license, missing := parseLicenseManual(fileDir, manualLicense)
	if missing {
		lDir, lType, missing := parseLicenseAuto(dir, fileDir)
		lDir = lDir[len(dir)+1:]
		if missing {
			log.Println("Could not find license for ", lDir)
			exitOnMissing = true
		}
		if lType != "" {
			arr := licenseMap[lType]
			if !InStringSlice(arr, lDir) {
				arr = append(arr, lDir)
				licenseMap[lType] = arr
			}
		}
	} else if len(license) > 0 {
		//License can be either a single word, then we will check in the licenseMap
		//If it is more than one word, we will simply place it there ...
		if strings.Index(license, " ") == -1 {
			arr, exists := licenseMap[license]
			if exists {
				if !InStringSlice(arr, lDir) {
					arr = append(arr, lDir)
					licenseMap[license] = arr
				}
			} else {
				foundManualLicense[lDir] = license
			}
		} else {
			foundManualLicense[lDir] = license
		}
	}
}

func generateLicenseFile(lTypeMap map[string][]string, lContentMap map[string]string) string {
	licenseMap := initLicenseMap()
	res := ""
	for k, v := range lTypeMap {
		fullLicense := licenseMap[k]
		projects := ""
		for _, p := range v {
			projects += p + "\n"
		}
		projects += fullLicense + "\n"
		res += projects
	}
	for project, fullLicense := range lContentMap {
		res += project + "\n" + fullLicense + "\n"
	}
	return res
}

// InStringSlice checks if val string is in s slice, case insensitive.
func InStringSlice(slice []string, val string) bool {
	for _, v := range slice {
		if strings.EqualFold(v, val) {
			return true
		}
	}
	return false
}

func parseLicenseAuto(dir, fileDir string) (lDir string, lType string, missing bool) {
	// This case will work if there is a guessable license file in the
	// current working directory.
	dirs := strings.Split(fileDir, "/")
	currentDir := dir
	missing = true
	lDir = filepath.Join(dir, fileDir)
	for i := range dirs {
		currentDir = filepath.Join(currentDir, dirs[i])
		l, err := license.NewFromDir(currentDir)
		if err != nil {
			continue
		}
		missing = false
		lType = l.Type
		lDir = currentDir
		break
	}
	return
}

func prepareManualLicense(vendorDir string) map[string]string {
	fileName := filepath.Join(vendorDir, "manualLicense.json")
	log.Println("Processing manual license file: ", fileName)
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Println("No manual license file")
		return map[string]string{}
	}
	licenseMap := map[string]string{}
	err = json.Unmarshal(data, &licenseMap)
	return licenseMap
}

//parseLicenseManual will look for the manual license file index, to add files that cannot be found automatically
func parseLicenseManual(dir string, manualFileMap map[string]string) (lDir string, lContent string, missing bool) {
	dirs := strings.Split(dir, "/")
	currentDir := ""
	missing = true
	lDir = dir
	for i := range dirs {
		currentDir = filepath.Join(currentDir, dirs[i])
		content, exists := manualFileMap[currentDir]
		if exists {
			missing = false
			lContent = content
			lDir = currentDir
			break
		}
	}
	return
}
