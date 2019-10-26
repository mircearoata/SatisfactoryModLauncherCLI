package smlhandler

import (
	"C"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/Masterminds/semver"

	"github.com/mircearoata/SatisfactoryModLauncherCLI/paths"
	"github.com/mircearoata/SatisfactoryModLauncherCLI/util"
)

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procLoadLibraryExA = modkernel32.NewProc("LoadLibraryExA")
	procFreeLibrary    = modkernel32.NewProc("FreeLibrary")
	procGetProcAddress = modkernel32.NewProc("GetProcAddress")
)

const smlGitHubReleasesAPIurl = "https://api.github.com/repos/satisfactorymodding/SatisfactoryModLoader/releases"

var oldVersionsChecksum map[string]string = map[string]string{
	"v1.0.0-pr1": "af8f291c9f9534fb0972e976d9e87807126ec7976fd1eb32af9438e34cb0316d",
	"v1.0.0-pr2": "ce0e923f44623626dc500138bf400f030f2175765f7dd33aa84b9611bf36ca1b",
	"v1.0.0-pr3": "424a347308da025e99d6210ba6379a0487bdd01561423f071044574799aa65e6",
	"v1.0.0-pr4": "251d2798fc3d143f6cfd5bc9c73a37d8663add2ce4f57f6d8f19512ef8c8df65",
	"v1.0.0-pr5": "ac09dc25d32bc00a7bd9da4bc9bd10cc4d49229088c1d32de91fcdf24639ed87",
	"v1.0.0-pr6": "c2bef7b4cda4b7e268741e68e59ea737642f18316379632b52b7ba5d1e140855",
	"v1.0.0-pr7": "66e1fc34e08eba6920cbbe1eff8e8948821ca916383b790e6e1b18417cba6e1d",
	"1.0.0":      "29ce7f569ae30c62758adf4dead521b1b16433192f280ab62b59fd8f6dc0e8c7",
	"1.0.1":      "d15894a93db6a14d3c036a9e0f1da5d6e4b97e94f25374305b7ffdbcd3a5ebd9",
	// SML version is exported since SML 1.0.2
}

// GetInstalledVersion gets the version of the SML dll (0.0.0 = not found, 0.0.1 = <1.0.2 and not official)
func GetInstalledVersion(satisfactoryPath string) string {
	dllPath := path.Join(satisfactoryPath, "xinput1_3.dll")
	if !paths.Exists(dllPath) {
		return "0.0.0"
	}
	dllPathNullTerminated := append([]byte(dllPath), 0)
	dll, _, loadErr := procLoadLibraryExA.Call(uintptr(unsafe.Pointer(&dllPathNullTerminated[0])), uintptr(unsafe.Pointer(nil)), 1)
	if loadErr != syscall.Errno(0x0) {
		util.Check(loadErr)
	}
	defer procFreeLibrary.Call(dll)
	smlVersionString := "smlVersion"
	smlVersionStringNullTerminated := append([]byte(smlVersionString), 0)
	smlVersion, _, getProcErr := procGetProcAddress.Call(dll, uintptr(unsafe.Pointer(&smlVersionStringNullTerminated[0])))
	if getProcErr != syscall.Errno(0x0) { // happens when using an old version of SML which doesn't export the version, fallback to hashes
		fileHash := util.Sha256File(dllPath)
		for k, v := range oldVersionsChecksum {
			if v == fileHash {
				return k
			}
		}
		return "0.0.1"
	}
	smlVersionFinal := C.GoString((*C.char)(unsafe.Pointer(smlVersion)))
	return smlVersionFinal
}

// SMLAsset part of GitHub asset structure
type SMLAsset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
}

// SMLRelease part of GitHub release structure
type SMLRelease struct {
	Version         string     `json:"tag_name"`
	Description     string     `json:"body"`
	ReleaseDateTime time.Time  `json:"published_at"`
	Assets          []SMLAsset `json:"assets"`
	DownloadURL     string
}

// GetSMLReleases finds the versions of SML available to download from GitHub
func GetSMLReleases() []SMLRelease {
	response, httpErr := http.Get(smlGitHubReleasesAPIurl)
	util.Check(httpErr)
	body, _ := ioutil.ReadAll(response.Body)
	var releases []SMLRelease
	json.Unmarshal([]byte(string(body)), &releases)
	installInstructionsRegex, _ := regexp.Compile(`#\s*Installation(.+\s)*\n`)
	for i := 0; i < len(releases); i++ {
		if strings.HasPrefix(releases[i].Version, "v") {
			releases[i].Version = releases[i].Version[1:]
		}
		releases[i].Description = installInstructionsRegex.ReplaceAllString(releases[i].Description, "")
		for _, asset := range releases[i].Assets {
			if asset.Name == "xinput1_3.dll" {
				releases[i].DownloadURL = asset.DownloadURL
				break
			}
		}
	}
	sort.Slice(releases[:], func(i, j int) bool {
		return releases[i].ReleaseDateTime.Before(releases[j].ReleaseDateTime)
	})
	return releases
}

// GetLatestSML finds the latest version of SML available to download from GitHub
func GetLatestSML() SMLRelease {
	releases := GetSMLReleases()
	return releases[len(releases)-1]
}

func shouldInstall(satisfactoryPath string, version string) bool {
	installed, semverErr1 := semver.NewVersion(GetInstalledVersion(satisfactoryPath))
	util.Check(semverErr1)
	new, semverErr2 := semver.NewVersion(version)
	if semverErr2 != nil {
		return false // invalid semver
	}
	return installed.Compare(new) == -1
}

// InstallSML checks the versions of SML and installs if the specified version is newer than the installed version
func InstallSML(satisfactoryPath string, version string) error {
	if shouldInstall(satisfactoryPath, version) {
		releases := GetSMLReleases()
		for _, release := range releases {
			if release.Version == version {
				return util.DownloadFile(path.Join(satisfactoryPath, "xinput1_3.dll"), release.DownloadURL)
			}
		}
		return errors.New("SML version " + version + " does not exist")
	}
	return errors.New("SML installed version newer than target")
}

// UpdateSML finds the latest version of SML available to download from GitHub and updates to it if newer
func UpdateSML(satisfactoryPath string) error {
	version := GetLatestSML().Version
	if shouldInstall(satisfactoryPath, version) {
		releases := GetSMLReleases()
		for _, release := range releases {
			if release.Version == version {
				UninstallSML(satisfactoryPath)
				return util.DownloadFile(path.Join(satisfactoryPath, "xinput1_3.dll"), release.DownloadURL)
			}
		}
		return errors.New("SML version " + version + " does not exist")
	}
	return errors.New("SML already up to date")
}

// UninstallSML removes the SML dll from the path
func UninstallSML(satisfactoryPath string) error {
	dllPath := path.Join(satisfactoryPath, "xinput1_3.dll")
	if !paths.Exists(dllPath) {
		return errors.New("SML is not installed at this path")
	}
	err := os.Remove(dllPath)
	return err
}
