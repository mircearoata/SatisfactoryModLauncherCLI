package satisfactoryinstall

import (
	"encoding/json"
	"io/ioutil"
	"path"

	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

type SatisfactoryInstall struct {
	Name             string
	Version          string
	Path             string
	LaunchExecutable string
}

const EpicGamesManifestsPath = `C:\ProgramData\Epic\EpicGamesLauncher\Data\Manifests`

var SatisfactoryVersions []SatisfactoryInstall = []SatisfactoryInstall{}

func FindSatisfactoryInstalls() {
	files, err := ioutil.ReadDir(EpicGamesManifestsPath)
	util.Check(err)
	for _, manifestFile := range files {
		if manifestFile.IsDir() {
			continue
		}
		manifestContent, err2 := ioutil.ReadFile(path.Join(EpicGamesManifestsPath, manifestFile.Name()))
		util.Check(err2)
		var manifest map[string]interface{}
		json.Unmarshal([]byte(manifestContent), &manifest)
		if manifest["CatalogNamespace"] == "crab" {
			SatisfactoryVersions = append(SatisfactoryVersions, SatisfactoryInstall{manifest["DisplayName"].(string), manifest["AppVersionString"].(string), manifest["InstallLocation"].(string), manifest["LaunchExecutable"].(string)})
		}
	}
}
