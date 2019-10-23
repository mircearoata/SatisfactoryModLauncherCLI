package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mircearoata/SatisfactoryModLauncherCLI/ficsitapp"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/modhandler"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

const SMLAUNCHER_VERSION = "0.0.1"

var args []string

func paramNeeded(param string) {
	log.Fatalln(param + " is needed")
}

func readParam(name string, needed bool) string {
	if len(args) == 0 {
		if needed {
			paramNeeded(name)
		}
		return ""
	}
	ret := args[0]
	args = args[1:]
	return ret
}

func readParamDefault(name string, defaultValue string) string {
	val := readParam(name, false)
	if val == "" {
		return defaultValue
	}
	return val
}

func initSMLauncher() {
	paths.Init()
	ficsitapp.Init()
}

func main() {
	initSMLauncher()

	args = os.Args[1:]
	commandName := readParam("command", false)
	if len(commandName) == 0 {
		fmt.Println("Available commands: download, remove, update, version")
		return
	}
	if commandName == "download" || commandName == "remove" || commandName == "update" || commandName == "list_versions" {
		modID := readParam("modID", true)
		version := readParam("modVersion", false)
		if commandName == "download" {
			if len(version) == 0 {
				version = ficsitapp.GetLatestModVersion(modID)
			}
			success, err := ficsitapp.DownloadModVersion(modID, version)
			util.Check(err)
			if success {
				fmt.Println("Downloaded " + modID + "@" + version)
			} else {
				fmt.Println("Mod " + modID + "@" + version + " could not be downloaded")
			}
		} else if commandName == "remove" {
			if len(version) == 0 {
				downloadedVersions, err := modhandler.GetDownloadedModVersions(modID)
				util.Check(err)
				for _, modVersion := range downloadedVersions {
					if !modhandler.Remove(modID, modVersion) {
						fmt.Println("Failed to remove " + modID + "@" + modVersion)
					}
				}
			} else {
				if !modhandler.Remove(modID, version) {
					fmt.Println("Failed to remove " + modID + "@" + version)
				}
			}
		} else if commandName == "update" {
			updated := modhandler.Update(modID)
			currentVersion, err := modhandler.GetLatestDownloadedVersion(modID)
			util.Check(err)
			if updated {
				fmt.Println("Updated " + modID + " to " + currentVersion)
			} else {
				fmt.Println(modID + " is already up to date (" + currentVersion + ")")
			}
		} else if commandName == "list_versions" {
			modVersions, _ := modhandler.GetDownloadedModVersions(modID)
			fmt.Println(strings.Join(modVersions, ", "))
		}
	} else if commandName == "install" || commandName == "uninstall" {
		satisfactoryPath := readParam("satisfactoryPath (ending in Binaries/Win64)", true)
		modID := readParam("modID", true)
		version := readParam("modVersion", false)
		if len(version) == 0 {
			var err error
			version, err = modhandler.GetLatestDownloadedVersion(modID)
			util.Check(err)
		}
		if !paths.Exists(satisfactoryPath) {
			log.Fatalln(errors.New("Invalid Satisfactory path"))
		}
		if commandName == "install" {
			modhandler.Install(modID, version, satisfactoryPath)
		} else if commandName == "uninstall" {
			modhandler.Uninstall(modID, version, satisfactoryPath)

		}
	} else if commandName == "version" {
		fmt.Println(SMLAUNCHER_VERSION)
	} else {
		fmt.Println("Unrecognized command \"" + commandName + "\"")
	}
}
