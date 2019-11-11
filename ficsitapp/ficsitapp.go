package ficsitapp

import (
	"context"
	"errors"
	"log"
	"path"
	"sort"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/machinebox/graphql"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

const baseAPI = `https://api.ficsit.app`
const ficsitappAPI = baseAPI + `/v2/query`

var api *graphql.Client = graphql.NewClient(ficsitappAPI)

const modVersionLatestRequest = `
query($modID: ModID!){
	getMod(modId: $modID)
	{
		latestVersions
		{
			alpha
			{
				version
			}
			beta
			{
				version
			}
			release
			{
				version
			}
		}
	}
}
`

const modVersionsRequest = `
query($modID: ModID!){
	getMod(modId: $modID)
	{
		versions
		{
			version,
			stability
		}
	}
}
`

const modVersionDownloadLinkRequest = `
query($modID: ModID!, $version: String!){
	getMod(modId: $modID)
	{
		version(version: $version)
		{
			link
		}
	}
}
`

var availableVersionStabilities = []string{"alpha", "beta", "release"}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

// ModVersion from ficsit.app
type ModVersion struct {
	Version   string
	Stability string
}

// GetModVersions gets the versions of the mod
func GetModVersions(modID string) []ModVersion {
	req := graphql.NewRequest(modVersionsRequest)
	req.Var("modID", modID)
	ctx := context.Background()
	var respData map[string]interface{}
	apiErr := api.Run(ctx, req, &respData)
	util.Check(apiErr)
	if respData["getMod"] == nil {
		log.Fatalln("Mod " + modID + " does not exist")
	}
	versions := (respData["getMod"].(map[string]interface{})["versions"]).([]interface{})
	structVersions := []ModVersion{}
	for _, version := range versions {
		structVersion := ModVersion{version.(map[string]interface{})["version"].(string), version.(map[string]interface{})["stability"].(string)}
		structVersions = append(structVersions, structVersion)
	}
	sort.Slice(versions, func(i, j int) bool {
		verA, errA := semver.NewVersion(structVersions[i].Version)
		verB, errB := semver.NewVersion(structVersions[j].Version)
		if errA != nil {
			return false
		}
		if errB != nil {
			return true
		}
		return verA.Compare(verB) == -1
	})
	return structVersions
}

// GetLatestModVersion gets the latest version of the mod
func GetLatestModVersion(modID string) string {
	req := graphql.NewRequest(modVersionLatestRequest)
	req.Var("modID", modID)
	ctx := context.Background()
	var respData map[string]interface{}
	apiErr := api.Run(ctx, req, &respData)
	util.Check(apiErr)
	if respData["getMod"] == nil {
		log.Fatalln("Mod " + modID + " does not exist")
	}
	mod := respData["getMod"].(map[string]interface{})
	latestVersions := mod["latestVersions"].(map[string]interface{})
	versions := []string{}
	for _, versionStability := range availableVersionStabilities {
		if latestVersions[versionStability] != nil {
			versionNumber := latestVersions[versionStability].(map[string]interface{})["version"]
			if versionNumber != nil && versionNumber != "" {
				versions = append(versions, versionNumber.(string))
			}
		}
	}
	if len(versions) == 0 {
		log.Fatalln("Mod " + modID + " has no available version")
	}
	sort.Slice(versions, func(i, j int) bool {
		verA, errA := semver.NewVersion(versions[i])
		verB, errB := semver.NewVersion(versions[j])
		if errA != nil {
			return false
		}
		if errB != nil {
			return true
		}
		return verA.Compare(verB) == -1
	})
	return versions[len(versions)-1]
}

// DownloadModVersion downloads the specified version of the mod
func DownloadModVersion(modID string, version string) (bool, error) {
	req := graphql.NewRequest(modVersionDownloadLinkRequest)
	req.Var("modID", modID)
	req.Var("version", version)
	ctx := context.Background()
	var respData map[string]interface{}
	apiErr := api.Run(ctx, req, &respData)
	util.Check(apiErr)
	if respData["getMod"] == nil {
		log.Fatalln("Mod " + modID + " does not exist")
	}
	versionResponse := respData["getMod"].(map[string]interface{})["version"]
	if versionResponse == nil {
		// try with prefix v
		vSuccess, vError := DownloadModVersion(modID, "v"+version)
		if !vSuccess {
			return false, errors.New("Mod " + modID + " has no version " + version)
		}
		return vSuccess, vError
	}
	link := baseAPI + versionResponse.(map[string]interface{})["link"].(string)
	downloadErr := util.DownloadFile(path.Join(paths.ModDir(modID), modID+"_"+version+".zip"), link)
	if downloadErr != nil {
		return false, downloadErr
	}
	return true, nil
}

// GetModFromVersionConstraint returns the latest mod version which meets a constraint
func GetModFromVersionConstraint(modID string, versionConstraint string) (string, error) {
	version := ""
	constraint, constraintErr := semver.NewConstraint(versionConstraint)
	util.Check(constraintErr)
	for _, availableVersion := range GetModVersions(modID) {
		ver, verErr := semver.NewVersion(availableVersion.Version)
		util.Check(verErr)
		if constraint.Check(ver) {
			version = availableVersion.Version
			if strings.HasPrefix(version, "v") {
				version = version[1:]
			}
			return version, nil
		}
	}
	return "", errors.New("No version of mod " + modID + " matched constraint " + versionConstraint)
}

// DownloadModLatest downloads the latest version of the mod
func DownloadModLatest(modID string) (bool, error) {
	return DownloadModVersion(modID, GetLatestModVersion(modID))
}
