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
	"github.com/mircearoata/SatisfactoryModLauncherCLI/smlhandler"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

const smlauncherVersion = "0.0.1"

const helpMessage = `
Satisfactory Mod Launcher CLI
Commands: 
	help - displays this help message
	download - download a mod from https://ficsit.app by its id and version (optional, defaults to newest)
	remove - deletes a downloaded mod
	update - downloads the newest version of the mod and deletes the old ones
	check_updates - checks for available new versions of mods and SML
	install - installs the mod to the Satisfactory install
	uninstall - removes the mod from the Satisfactory install
	install_sml - installs SML
	uninstall_sml - uninstalls SML
	update_sml - updates SML
	sml_version - shows the installed version of SML
	list_versions - shows the list of downloaded versions of a mod
	list - shows the installed mods list and their version
	list_installed - shows the installed mods
	mods_dir - shows the directory where SMLauncher downloads the mods
	version - shows the Satisfactory Mod Launcher CLI version
`

var args []string

func initSMLauncher() {
	paths.Init()
}

func main() {
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	initSMLauncher()
	args = os.Args[1:]
	if len(args) == 0 {
		log.Println(helpMessage)
		return
	}
	commandName := os.Args[1]
	parser := argparse.NewParser("SatisfactoryModLauncher CLI", "Handles mod download and install")
	if commandName == "help" {
		log.Println(helpMessage)
	} else if commandName == "download" || commandName == "remove" || commandName == "update" || commandName == "list_versions" {
		modIDParam := parser.String("m", "mod", &argparse.Options{Required: true, Help: "ficsit.app mod ID"})
		versionParam := parser.String("v", "version", &argparse.Options{Required: false, Help: "mod version"})
		parseErr := parser.Parse(args)
		util.Check(parseErr)
		modID := *modIDParam
		version := *versionParam
		if commandName == "download" {
			if len(version) == 0 {
				version = ficsitapp.GetLatestModVersion(modID)
			}
			success, downloadErr := ficsitapp.DownloadModVersion(modID, version)
			util.Check(downloadErr)
			if success {
				fmt.Println("Downloaded " + modID + "@" + version)
			} else {
				fmt.Println("Mod " + modID + "@" + version + " could not be downloaded")
			}
		} else if commandName == "remove" {
			if len(version) == 0 {
				downloadedVersions, getDownloadedErr := modhandler.GetDownloadedModVersions(modID)
				util.Check(getDownloadedErr)
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
			currentVersion, getLatestDownloadedErr := modhandler.GetLatestDownloadedVersion(modID)
			util.Check(getLatestDownloadedErr)
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
		modIDParam := parser.String("m", "mod", &argparse.Options{Required: true, Help: "ficsit.app mod ID"})
		versionParam := parser.String("v", "version", &argparse.Options{Required: false, Help: "mod version"})
		satisfactoryPathParam := parser.String("p", "path", &argparse.Options{Required: true, Help: "satisfactory install path (ending in Binaries/Win64)"})
		parseErr := parser.Parse(args)
		util.Check(parseErr)
		modID := *modIDParam
		version := *versionParam
		satisfactoryPath := *satisfactoryPathParam
		if len(version) == 0 {
			var getLatestErr error
			version, getLatestErr = modhandler.GetLatestDownloadedVersion(modID)
			util.Check(getLatestErr)
		}
		if !paths.Exists(satisfactoryPath) {
			log.Fatalln(errors.New("Invalid Satisfactory path"))
		}
		if commandName == "install" {
			modhandler.Install(modID, version, satisfactoryPath)
		} else if commandName == "uninstall" {
			modhandler.Uninstall(modID, version, satisfactoryPath)
		}
	} else if commandName == "list" {
		mods := modhandler.GetDownloadedMods()
		for _, mod := range mods {
			fmt.Println(mod.Name + " (" + mod.ModID + ")" + " - " + mod.Version)
		}
	} else if commandName == "list_installed" {
		satisfactoryPathParam := parser.String("p", "path", &argparse.Options{Required: true, Help: "satisfactory install path (ending in Binaries/Win64)"})
		parseErr := parser.Parse(args)
		util.Check(parseErr)
		satisfactoryPath := *satisfactoryPathParam
		mods := modhandler.GetInstalledMods(satisfactoryPath)
		for _, mod := range mods {
			fmt.Println(mod.Name + " (" + mod.ModID + ")" + " - " + mod.Version)
		}
	} else if commandName == "install_sml" || commandName == "uninstall_sml" || commandName == "update_sml" || commandName == "sml_version" {
		satisfactoryPathParam := parser.String("p", "path", &argparse.Options{Required: true, Help: "satisfactory install path (ending in Binaries/Win64)"})
		if commandName == "sml_version" {
			parseErr := parser.Parse(args)
			util.Check(parseErr)
			satisfactoryPath := *satisfactoryPathParam
			fmt.Println(smlhandler.GetInstalledVersion(satisfactoryPath))
		} else if commandName == "install_sml" {
			smlVersionParam := parser.String("v", "version", &argparse.Options{Required: false, Help: "SML version"})
			parseErr := parser.Parse(args)
			util.Check(parseErr)
			smlVersion := *smlVersionParam
			satisfactoryPath := *satisfactoryPathParam
			if smlVersion == "" {
				smlVersion = smlhandler.GetLatestSML().Version
			}
			installErr := smlhandler.InstallSML(satisfactoryPath, smlVersion)
			util.Check(installErr)
			fmt.Println("Installed SML@" + smlVersion)
		} else if commandName == "update_sml" {
			parseErr := parser.Parse(args)
			util.Check(parseErr)
			satisfactoryPath := *satisfactoryPathParam
			updateErr := smlhandler.UpdateSML(satisfactoryPath)
			util.Check(updateErr)
			fmt.Println("Updated to SML@" + smlhandler.GetInstalledVersion(satisfactoryPath))
		} else if commandName == "uninstall_sml" {
			parseErr := parser.Parse(args)
			util.Check(parseErr)
			satisfactoryPath := *satisfactoryPathParam
			uninstallErr := smlhandler.UninstallSML(satisfactoryPath)
			util.Check(uninstallErr)
			fmt.Println("Uninstalled SML")
		}
	} else if commandName == "check_updates" {
		satisfactoryPathParam := parser.String("p", "path", &argparse.Options{Required: false, Help: "satisfactory install path (ending in Binaries/Win64)"})
		autoInstallParam := parser.Flag("i", "install", &argparse.Options{Required: false, Help: "Automatically download and install the updates"})
		parseErr := parser.Parse(args)
		util.Check(parseErr)
		satisfactoryPath := *satisfactoryPathParam
		autoInstall := *autoInstallParam
		hasModUpdates := modhandler.CheckForUpdates(autoInstall)
		hasSMLUpdates := false
		if satisfactoryPath != "" {
			hasSMLUpdates = smlhandler.CheckForUpdates(satisfactoryPath, autoInstall)
		}
		if !hasModUpdates && !hasSMLUpdates {
			fmt.Println("Already up to date")
		}
	} else if commandName == "mods_dir" {
		fmt.Println(paths.ModsDir)
	} else if commandName == "version" {
		fmt.Println(smlauncherVersion)
	} else {
		log.Println("Unrecognized command \"" + commandName + "\"")
	}
}
