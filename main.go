package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/akamensky/argparse"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/ficsitapp"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/modhandler"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

const SMLAUNCHER_VERSION = "0.0.1"

const helpMessage = `
Satisfactory Mod Launcher CLI
Commands: 
	help - displays this help message
	download - download a mod from https://ficsit.app by its id and version (optional, defaults to newest)
	remove - deletes a downloaded mod
	update - downloads the newest version of the mod and deletes the old ones
	install - installs the mod to the Satisfactory install
	uninstall - removes the mod from the Satisfactory install
	list_versions - shows the list of downloaded versions of a mod
	version - shows the Satisfactory Mod Launcher CLI version
`

var args []string

func initSMLauncher() {
	paths.Init()
	ficsitapp.Init()
}

func main() {
	initSMLauncher()
	args = os.Args[1:]
	if len(args) == 0 {
		fmt.Println(helpMessage)
		return
	}
	commandName := os.Args[1]
	parser := argparse.NewParser("SatisfactoryModLauncher CLI", "Handles mod download and install")
	modID_param := parser.String("m", "mod", &argparse.Options{Required: true, Help: "ficsit.app mod ID"})
	version_param := parser.String("v", "version", &argparse.Options{Required: false, Help: "mod version"})
	if commandName == "help" {
		fmt.Println(helpMessage)
	} else if commandName == "download" || commandName == "remove" || commandName == "update" || commandName == "list_versions" {
		err := parser.Parse(args)
		util.Check(err)
		modID := *modID_param
		version := *version_param
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
		satisfactoryPath_param := parser.String("p", "path", &argparse.Options{Required: true, Help: "satisfactory install path (ending in Binaries/Win64)"})
		err := parser.Parse(args)
		util.Check(err)
		satisfactoryPath := *satisfactoryPath_param
		modID := *modID_param
		version := *version_param
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
