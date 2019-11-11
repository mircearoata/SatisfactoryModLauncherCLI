package satisfactoryinstall

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

// SatisfactoryInstall part of Epic Games' manifest
type SatisfactoryInstall struct {
	Name             string
	Version          string
	Path             string
	LaunchExecutable string
}

const epicGamesManifestsPath = `C:\ProgramData\Epic\EpicGamesLauncher\Data\Manifests`

// SatisfactoryVersions keep track of the SF install dirs
var SatisfactoryVersions []SatisfactoryInstall = []SatisfactoryInstall{}

// TODO: support version names (EA/EXP) instead of full paths
// TODO: find where the manifests are stored in other OSs
// Actually, this ^ won't work for dedicated servers, so exact paths are still needed

// FindSatisfactoryInstalls checks Epic Games' manifests for SF install dirs
func FindSatisfactoryInstalls() {
	files, err := ioutil.ReadDir(epicGamesManifestsPath)
	util.Check(err)
	for _, manifestFile := range files {
		if manifestFile.IsDir() {
			continue
		}
		manifestContent, err2 := ioutil.ReadFile(path.Join(epicGamesManifestsPath, manifestFile.Name()))
		util.Check(err2)
		var manifest map[string]interface{}
		json.Unmarshal([]byte(manifestContent), &manifest)
		if manifest["CatalogNamespace"] == "crab" {
			SatisfactoryVersions = append(SatisfactoryVersions, SatisfactoryInstall{manifest["DisplayName"].(string), manifest["AppVersionString"].(string), manifest["InstallLocation"].(string), manifest["LaunchExecutable"].(string)})
		}
	}
}
