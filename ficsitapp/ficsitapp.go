package ficsitapp

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"sort"

	"github.com/machinebox/graphql"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

const base_api = `https://api.ficsit.app`
const ficsitapp_api = base_api + `/v2/query`

var api *graphql.Client = graphql.NewClient(ficsitapp_api)

func Init() {
}

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

type ModVersion struct {
	Version   string
	Stability string
}

func GetModVersions(modID string) []ModVersion {
	req := graphql.NewRequest(modVersionsRequest)
	req.Var("modID", modID)
	ctx := context.Background()
	var respData map[string]interface{}
	err := api.Run(ctx, req, &respData)
	util.Check(err)
	versions := respData["getMod"].(map[string]interface{})["versions"].([]ModVersion)
	return versions
}

func GetLatestModVersion(modID string) string {
	req := graphql.NewRequest(modVersionLatestRequest)
	req.Var("modID", modID)
	ctx := context.Background()
	var respData map[string]interface{}
	err := api.Run(ctx, req, &respData)
	util.Check(err)
	latestVersions := respData["getMod"].(map[string]interface{})["latestVersions"].(map[string]interface{})
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
	sort.Strings(versions)
	return versions[len(versions)-1]
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func DownloadModVersion(modID string, version string) (bool, error) {
	req := graphql.NewRequest(modVersionDownloadLinkRequest)
	req.Var("modID", modID)
	req.Var("version", version)
	ctx := context.Background()
	var respData map[string]interface{}
	err := api.Run(ctx, req, &respData)
	util.Check(err)
	versionResponse := respData["getMod"].(map[string]interface{})["version"]
	if versionResponse == nil {
		return false, errors.New("Mod " + modID + " has no version " + version)
	}
	link := base_api + versionResponse.(map[string]interface{})["link"].(string)
	err = DownloadFile(path.Join(paths.ModDir(modID), modID+"_"+version+".zip"), link)
	if err != nil {
		return false, err
	}
	return true, nil
}

func DownloadModLatest(modID string) (bool, error) {
	return DownloadModVersion(modID, GetLatestModVersion(modID))
}
