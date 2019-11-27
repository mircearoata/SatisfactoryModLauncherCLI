package modhandler

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/Masterminds/semver"

	"github.com/mircearoata/SatisfactoryModLauncherCLI/ficsitapp"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

// ModFile contains the data.json objects information
type ModFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// DataJSON contains the data.json information
type DataJSON struct {
	ModID           string            `json:"mod_id"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	Authors         []string          `json:"authors"`
	Objects         []ModFile         `json:"objects"`
	Dependencies    map[string]string `json:"dependencies"`
	OptDependencies map[string]string `json:"optional_dependencies"`
}

// GetDataFromZip returns the data.json file in the zip
func GetDataFromZip(zipFileName string) DataJSON {
	zipFile, zipErr := zip.OpenReader(zipFileName)
	util.Check(zipErr)
	defer zipFile.Close()
	for _, file := range zipFile.File {
		if file.Name == "data.json" {
			fileContent := util.ReadAllFromZip(file)
			var data DataJSON
			json.Unmarshal(fileContent, &data)
			if strings.HasPrefix(data.Version, "v") {
				data.Version = data.Version[1:]
			}
			return data
		}
	}
	log.Fatalln(zipFileName + " does not contain a data.json. Contact the mod author.")
	return DataJSON{}
}

func getModZips(modID string) []string {
	modPath := paths.ModDir(modID)
	files, listDirErr := ioutil.ReadDir(modPath)
	if listDirErr != nil {
		return []string{}
	}
	zipFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".zip") {
			zipFiles = append(zipFiles, path.Join(modPath, file.Name()))
		}
	}
	return zipFiles
}

func findModZip(modID string, modVersion string) string {
	modZips := getModZips(modID)
	for _, file := range modZips {
		modData := GetDataFromZip(file)
		if modData.Version == modVersion {
			return file
		}
	}
	log.Fatalln("Mod " + modID + "@" + modVersion + " not found")
	return ""
}

// GetDownloadedModVersions Returns the downloaded versions of the mod
func GetDownloadedModVersions(modID string) ([]string, error) {
	versions := []string{}
	modZips := getModZips(modID)
	if len(modZips) == 0 {
		return []string{}, errors.New("Mod " + modID + " not downloaded")
	}
	for _, file := range modZips {
		modData := GetDataFromZip(file)
		versions = append(versions, modData.Version)
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return []string{}, errors.New("No version of " + modID + " is downloaded")
	}
	return versions, nil
}

// GetDownloadedMods returns all mods found in the smlauncher mods dir
func GetDownloadedMods() []DataJSON {
	modPath := paths.ModsDir
	files, listDirErr := ioutil.ReadDir(modPath)
	util.Check(listDirErr)
	mods := []DataJSON{}
	for _, file := range files {
		if file.IsDir() {
			modZips := getModZips(file.Name())
			for _, modZip := range modZips {
				mods = append(mods, GetDataFromZip(modZip))
			}
		}
	}
	return mods
}

// GetInstalledMods returns all mods found in the sml mods dir
func GetInstalledMods(smlPath string) []DataJSON {
	smlModsDir := path.Join(smlPath, "mods")
	files, listDirErr := ioutil.ReadDir(smlModsDir)
	util.Check(listDirErr)
	zipFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".zip") {
			zipFiles = append(zipFiles, path.Join(smlModsDir, file.Name()))
		}
	}
	mods := []DataJSON{}
	for _, zipFile := range zipFiles {
		mods = append(mods, GetDataFromZip(zipFile))
	}
	return mods
}

// GetLatestDownloadedVersion Returns the latest downloaded version of the mod
func GetLatestDownloadedVersion(modID string) (string, error) {
	versions, getDownloadedErr := GetDownloadedModVersions(modID)
	if len(versions) == 0 {
		return "", getDownloadedErr
	}
	return versions[len(versions)-1], getDownloadedErr
}

// GetDependencies returns the non optional dependencies of a mod
func GetDependencies(modID string, modVersion string) map[string]string {
	data := GetDataFromZip(findModZip(modID, modVersion))
	return data.Dependencies
}

// Remove Removes the mod file from the downloaded mods
func Remove(modID string, modVersion string) bool {
	modZip := findModZip(modID, modVersion)
	if modZip == "" {
		return false
	}
	removeErr := os.Remove(modZip)
	util.Check(removeErr)
	dirEmpty, _ := paths.IsEmpty(paths.ModDir(modID))
	if dirEmpty {
		removeErr := os.Remove(paths.ModDir(modID))
		util.Check(removeErr)
	}
	return true
}

func shouldDownloadUpdate(oldVersion string, updateVersion string) bool {
	old, oldErr := semver.NewVersion(oldVersion)
	util.Check(oldErr)
	new, newErr := semver.NewVersion(updateVersion)
	util.Check(newErr)
	return old.Compare(new) == -1
}

// Update Tries to update the mod. Returns true if the mod was updated, false if the local file is already up to date
func Update(modID string) (bool, int) {
	ficsitAppModVersion := ficsitapp.GetLatestModVersion(modID)
	localModVersion, getLatestDownloadedErr := GetLatestDownloadedVersion(modID)
	util.Check(getLatestDownloadedErr)
	if ficsitAppModVersion != localModVersion {
		modVersions, getDownloadedErr := GetDownloadedModVersions(modID)
		util.Check(getDownloadedErr)
		for _, modVersion := range modVersions {
			modFile := findModZip(modID, modVersion)
			removeErr := os.Remove(modFile)
			util.Check(removeErr)
		}
		success, dependencyCnt := DownloadModWithDependencies(modID, ficsitapp.GetLatestModVersion(modID))
		return success, dependencyCnt
	}
	return false, 0
}

// Install the mod to the SML path
func Install(modID string, modVersion string, smlPath string) bool {
	if IsModInstalled(modID, smlPath) {
		return false
	}
	smlModsDir := path.Join(smlPath, "mods")
	os.MkdirAll(smlModsDir, os.ModePerm)
	modZipPath := findModZip(modID, modVersion)
	copyErr := paths.CopyFile(modZipPath, path.Join(smlModsDir, path.Base(modZipPath)))
	util.Check(copyErr)
	return true
}

// Uninstall the mod from the SML path
func Uninstall(modID string, modVersion string, smlPath string) bool {
	smlModsDir := path.Join(smlPath, "mods")
	files, listDirErr := ioutil.ReadDir(smlModsDir)
	util.Check(listDirErr)
	zipFiles := []string{}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".zip") {
			zipFiles = append(zipFiles, path.Join(smlModsDir, file.Name()))
		}
	}
	for _, zipFile := range zipFiles {
		modData := GetDataFromZip(zipFile)
		if modData.ModID == modID && modData.Version == modVersion {
			err := os.Remove(zipFile)
			util.Check(err)
			return true
		}
	}
	log.Fatalln("Mod " + modID + "@" + modVersion + " is not installed")
	return false
}

// CheckForUpdates compares the installed version with the newest available and optionally downloads it
func CheckForUpdates(install bool) bool {
	downloadedMods := GetDownloadedMods()
	uniqueMods := []string{}
	for _, downloadedMod := range downloadedMods {
		if !util.Contains(uniqueMods, downloadedMod.ModID) {
			uniqueMods = append(uniqueMods, downloadedMod.ModID)
		}
	}
	hasUpdates := false
	for _, mod := range uniqueMods {
		latestVersion := ficsitapp.GetLatestModVersion(mod)
		downloadedVersion, _ := GetLatestDownloadedVersion(mod)
		hasUpdate := shouldDownloadUpdate(downloadedVersion, latestVersion)
		if hasUpdate {
			if install {
				Update(mod)
				fmt.Println("Updated " + mod + " to " + latestVersion)
			} else {
				fmt.Println(mod + "@" + latestVersion + " available")
			}
			hasUpdates = true
		}
	}
	return hasUpdates
}

// GetDownloadedModVersionWithConstraint returns the latest downloaded version that meets the constraint
func GetDownloadedModVersionWithConstraint(modID string, versionConstraint string) string {
	versions, _ := GetDownloadedModVersions(modID)
	constraint, constraintErr := semver.NewConstraint(versionConstraint)
	util.Check(constraintErr)
	for _, version := range versions {
		ver, err := semver.NewVersion(version)
		util.Check(err)
		if constraint.Check(ver) {
			return version
		}
	}
	return ""
}

// GetInstalledModVersions returns the data.jsons of the versions of the mod
func GetInstalledModVersions(modID string, smlPath string) []DataJSON {
	mods := GetInstalledMods(smlPath)
	modVersions := []DataJSON{}
	for _, mod := range mods {
		if mod.ModID == modID {
			modVersions = append(modVersions, mod)
		}
	}
	return modVersions
}

// IsModInstalled checks if a mod is installed
func IsModInstalled(modID string, smlPath string) bool {
	versions := GetInstalledModVersions(modID, smlPath)
	return len(versions) > 0
}

// IsModVersionWithConstraintInstalled checks if a mod is installed
func IsModVersionWithConstraintInstalled(modID string, versionConstraint string, smlPath string) bool {
	versions := GetInstalledModVersionWithConstraint(modID, versionConstraint, smlPath)
	return len(versions) > 0
}

// IsModVersionInstalled checks if a mod is installed
func IsModVersionInstalled(modID string, version string, smlPath string) bool {
	versions := GetInstalledModVersions(modID, smlPath)
	versionsString := []string{}
	fmt.Println(modID, version, versionsString)
	for _, ver := range versions {
		versionsString = append(versionsString, ver.Version)
	}
	fmt.Println(modID, version, versionsString)
	return util.Contains(versionsString, version)
}

// GetInstalledModVersionWithConstraint returns the latest installed version that meets the constraint
func GetInstalledModVersionWithConstraint(modID string, versionConstraint string, smlPath string) string {
	mods := GetInstalledModVersions(modID, smlPath)
	constraint, constraintErr := semver.NewConstraint(versionConstraint)
	util.Check(constraintErr)
	for _, modVersion := range mods {
		ver, err := semver.NewVersion(modVersion.Version)
		util.Check(err)
		if constraint.Check(ver) {
			return modVersion.Version
		}
	}
	return ""
}

// DownloadModWithDependencies downloads the mod and its dependencies
func DownloadModWithDependencies(modID string, version string) (bool, int) {
	success, downloadErr := ficsitapp.DownloadModVersion(modID, version)
	util.Check(downloadErr)
	if success {
		dependencyCnt := 0
		dependencies := GetDependencies(modID, version)
		for dependencyID, dependencyVersionConstraint := range dependencies {
			if GetDownloadedModVersionWithConstraint(dependencyID, dependencyVersionConstraint) == "" {
				depVersion, depErr := ficsitapp.GetModFromVersionConstraint(dependencyID, dependencyVersionConstraint)
				util.Check(depErr)
				dependencySuccess, depDepCnt := DownloadModWithDependencies(dependencyID, depVersion)
				if !dependencySuccess {
					success = false
					log.Println("Error downloading dependency " + dependencyID + "@" + dependencyVersionConstraint + " for mod " + modID + "@" + version)
				} else {
					dependencyCnt = dependencyCnt + depDepCnt
				}
			}
		}
		return success, dependencyCnt + 1
	}
	return false, 0
}

// InstallModWithDependencies installs the mod and its dependencies
func InstallModWithDependencies(modID string, version string, smlPath string) bool {
	success := Install(modID, version, smlPath)
	if success {
		dependencies := GetDependencies(modID, version)
		for dependencyID, dependencyVersionConstraint := range dependencies {
			if GetInstalledModVersionWithConstraint(dependencyID, dependencyVersionConstraint, smlPath) == "" {
				depVersion := GetDownloadedModVersionWithConstraint(dependencyID, dependencyVersionConstraint)
				if depVersion == "" {
					var depErr error
					depVersion, depErr = ficsitapp.GetModFromVersionConstraint(dependencyID, dependencyVersionConstraint)
					fmt.Println("Dependency " + dependencyID + "@" + dependencyVersionConstraint + " is not downloaded. Downloading " + dependencyID + "@" + depVersion)
					util.Check(depErr)
					downloadSuccess, _ := DownloadModWithDependencies(dependencyID, depVersion)
					if !downloadSuccess {
						log.Println("Error downloading dependency " + dependencyID + "@" + dependencyVersionConstraint + " for mod " + modID + "@" + version)
					}
				}
				dependencySuccess := InstallModWithDependencies(dependencyID, depVersion, smlPath)
				if !dependencySuccess {
					success = false
					log.Println("Error installing dependency " + dependencyID + "@" + dependencyVersionConstraint + " for mod " + modID + "@" + version)
				} else {
					fmt.Println("Installed dependency " + dependencyID + "@" + depVersion + " for mod " + modID + "@" + version)
				}
			}
		}
		return success
	}
	return false
}
