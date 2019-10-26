package modhandler

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sort"
	"strings"

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
	ModID       string    `json:"mod_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Authors     []string  `json:"authors"`
	Objects     []ModFile `json:"objects"`
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
		return []string{""}, errors.New("Mod " + modID + " not downloaded")
	}
	for _, file := range modZips {
		modData := GetDataFromZip(file)
		versions = append(versions, modData.Version)
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return []string{""}, errors.New("No version of " + modID + " is downloaded")
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
	return versions[len(versions)-1], getDownloadedErr
}

// Remove Removes the mod file from the downloaded mods
func Remove(modID string, modVersion string) bool {
	modZip := findModZip(modID, modVersion)
	if modZip == "" {
		return false
	}
	removeErr := os.Remove(modZip)
	util.Check(removeErr)
	return true
}

// Update Tries to update the mod. Returns true if the mod was updated, false if the local file is already up to date
func Update(modID string) bool {
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
		success, downloadErr := ficsitapp.DownloadModLatest(modID)
		util.Check(downloadErr)
		return success
	}
	return false
}

// Install the mod to the SML path
func Install(modID string, modVersion string, smlPath string) bool {
	smlModsDir := path.Join(smlPath, "mods")
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
