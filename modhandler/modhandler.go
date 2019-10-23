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

type ModFile struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type DataJson struct {
	ModID       string    `json:"mod_id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Authors     []string  `json:"authors"`
	Objects     []ModFile `json:"objects"`
}

func GetDataFromZip(zipFileName string) DataJson {
	zipFile, err := zip.OpenReader(zipFileName)
	util.Check(err)
	defer zipFile.Close()
	for _, file := range zipFile.File {
		if file.Name == "data.json" {
			fileContent := util.ReadAllFromZip(file)
			var data DataJson
			json.Unmarshal(fileContent, &data)
			return data
		}
	}
	log.Fatalln(zipFileName + " does not contain a data.json. Contact the mod author.")
	return DataJson{}
}

func getModZips(modID string) []string {
	modPath := path.Join(paths.ModsDir, modID)
	files, err := ioutil.ReadDir(modPath)
	if err != nil {
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
	return ""
}

// GetDownloadedModVersions Returns the downloaded versions of the mod
func GetDownloadedModVersions(modID string) ([]string, error) {
	versions := []string{}
	modZips := getModZips(modID)
	if len(modZips) == 0 {
		return []string{""}, errors.New("Mod " + modID + " not installed")
	}
	for _, file := range modZips {
		modData := GetDataFromZip(file)
		versions = append(versions, modData.Version)
	}
	sort.Strings(versions)
	if len(versions) == 0 {
		return []string{""}, errors.New("No version of " + modID + " is installed")
	}
	return versions, nil
}

// GetLatestDownloadedVersion Returns the latest downloaded version of the mod
func GetLatestDownloadedVersion(modID string) (string, error) {
	versions, err := GetDownloadedModVersions(modID)
	return versions[len(versions)-1], err
}

// Remove Removes the mod file from the downloaded mods
func Remove(modID string, modVersion string) bool {
	modZip := findModZip(modID, modVersion)
	if modZip == "" {
		return false
	}
	err := os.Remove(modZip)
	util.Check(err)
	return true
}

// Update Tries to update the mod. Returns true if the mod was updated, false if the local file is already up to date
func Update(modID string) bool {
	ficsitAppModVersion := ficsitapp.GetLatestModVersion(modID)
	localModVersion, err := GetLatestDownloadedVersion(modID)
	util.Check(err)
	if ficsitAppModVersion != localModVersion {
		modVersions, err := GetDownloadedModVersions(modID)
		util.Check(err)
		for _, modVersion := range modVersions {
			modFile := findModZip(modID, modVersion)
			err := os.Remove(modFile)
			util.Check(err)
		}
		ficsitapp.DownloadModLatest(modID)
		return true
	}
	return false
}

// Install Tries to install the mod to the SML path
func Install(modID string, modVersion string, smlPath string) bool {
	smlModsDir := path.Join(smlPath, "mods")
	modZipPath := findModZip(modID, modVersion)
	err := paths.CopyFile(modZipPath, path.Join(smlModsDir, path.Base(modZipPath)))
	util.Check(err)
	return true
}

// Uninstall Tries to uninstall the mod from the SML path
func Uninstall(modID string, modVersion string, smlPath string) bool {
	smlModsDir := path.Join(smlPath, "mods")
	modZipPath := findModZip(modID, modVersion)
	// TODO: The check checksum or something with all the zips in the mods folder because the user might change the file name
	smlModZipPath := path.Join(smlModsDir, path.Base(modZipPath))
	if paths.Exists(smlModZipPath) {
		err := os.Remove(smlModZipPath)
		util.Check(err)
		return true
	}
	return false
}
