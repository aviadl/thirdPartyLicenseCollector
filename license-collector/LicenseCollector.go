package licensecollector

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ryanuber/go-license"
)

//LicenseFileName is the default created license file name
const LicenseFileName = "THIRD_PARTY_LICENSE"

const vendorGoModuleFile = "modules.txt"

//licenseMissing indicates that a license is missing
var licenseMissing = false

//Collect collects licenses from npm and or go projects
func Collect(projectGO, projectNPM string, fileName string) error {
	licenseMap := map[string][]string{}
	foundManualLicense := map[string]string{}

	licenseMissing = false
	var err error
	if len(projectGO) > 0 {
		err = collectGoLicenseFiles(projectGO, licenseMap, foundManualLicense)
	}
	if len(projectNPM) > 0 {
		err = collectNpmLicenseFiles(projectNPM, licenseMap, foundManualLicense)
	}
	if err != nil {
		return err
	}
	if len(licenseMap)+len(foundManualLicense) == 0 {
		return errors.New("no licenses handled")
	}
	if licenseMissing {
		return errors.New("license missing")
	}
	fileData, err := generateLicenseFile(licenseMap, foundManualLicense)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, []byte(fileData), 0644)
	if err != nil {
		return err
	}
	log.Printf("generated license with name %s\n", fileName)
	return nil
}

func collectGoLicenseFiles(tmpGoDir string, licenseMap map[string][]string, foundManualLicense map[string]string) error {
	dir := filepath.Join(tmpGoDir, "vendor")
	log.Println("Go Project dir: ", dir)
	// test go modules
	fileName := filepath.Join(dir, vendorGoModuleFile)
	log.Println("Processing go module file: ", fileName)
	fileHandle, err := os.Open(fileName)
	if err != nil {
		log.Println(err)
		log.Printf("failed finding %s for third party packages. make sure you 'go mod vendor'\n", vendorGoModuleFile)
		return err
	}
	defer fileHandle.Close()

	packageMap := make(map[string]struct{})
	fileScanner := bufio.NewScanner(fileHandle)
	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		// take all packages.
		if strings.HasPrefix(line, "##") { // skip "## explicit" line which was added to modules.txt in GO 1.14
			continue
		}
		if strings.Index(line, "#") != 0 {
			continue
		}
		linePackage := strings.SplitN(line, " ", 3)[1]
		if len(linePackage) > 0 {
			packageMap[linePackage] = struct{}{}
		}
	}

	manualLicense := prepareManualLicense(tmpGoDir)
	for packagePath := range packageMap {
		doParseFile(dir, packagePath, manualLicense, licenseMap, foundManualLicense)
	}
	return nil
}

func collectNpmLicenseFiles(tmpNpmDir string, licenseMap map[string][]string, foundManualLicense map[string]string) error {
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
	lDir, licenseDescriptor, missing := parseLicenseManual(fileDir, manualLicense)
	if missing {
		lDir, lType, missing := parseLicenseAuto(dir, fileDir)
		lDir = lDir[len(dir)+1:]
		if missing {
			log.Println("Could not find license for ", lDir)
			licenseMissing = true
		}
		if lType != "" {
			arr := licenseMap[lType]
			if !InStringSlice(arr, lDir) {
				arr = append(arr, lDir)
				licenseMap[lType] = arr
			}
		}
	} else if len(licenseDescriptor) > 0 {
		//License can be either a single word, then we will check in the licenseMap
		//If it is more than one word, we will simply place it there ...
		if strings.Index(licenseDescriptor, " ") == -1 {
			arr, exists := licenseMap[licenseDescriptor]
			if exists {
				if !InStringSlice(arr, lDir) {
					arr = append(arr, lDir)
					licenseMap[licenseDescriptor] = arr
				}
			} else {
				foundManualLicense[lDir] = licenseDescriptor
			}
		} else {
			foundManualLicense[lDir] = licenseDescriptor
		}
	}
}

func generateLicenseFile(lTypeMap map[string][]string, lContentMap map[string]string) (string, error) {
	licenseMap := initLicenseMap()
	res := ""
	wrongLicense := map[string][]string{}
	for k, v := range lTypeMap {
		fullLicense, ok := licenseMap[k]
		if !ok {
			wrongLicense[k] = v
			continue
		}
		projects := ""
		for _, p := range v {
			projects += p + "\n"
		}
		projects += fullLicense + "\n"
		res += projects
	}
	if len(wrongLicense) > 0 {
		errMsg := "Wrong license files for the following libs"
		for k, v := range wrongLicense {
			errMsg += "\n " + k + ": " + fmt.Sprintf("%v", v)
		}
		return "", fmt.Errorf(errMsg)
	}
	for project, fullLicense := range lContentMap {
		res += project + "\n" + fullLicense + "\n"
	}
	return res, nil
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
