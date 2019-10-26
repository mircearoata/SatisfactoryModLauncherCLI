package util

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

// Check will print the error and exit
func Check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// ReadAllFromZip reads a zip file as bytes
func ReadAllFromZip(file *zip.File) []byte {
	fc, openErr := file.Open()
	Check(openErr)
	defer fc.Close()

	content, readErr := ioutil.ReadAll(fc)
	Check(readErr)

	return content
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, getErr := http.Get(url)
	if getErr != nil {
		return getErr
	}
	defer resp.Body.Close()

	// Create the file
	out, createErr := os.Create(filepath)
	if createErr != nil {
		return createErr
	}
	defer out.Close()

	// Write the body to file
	_, copyErr := io.Copy(out, resp.Body)
	return copyErr
}

// Sha256File calculates the checksum of the file
func Sha256File(path string) string {
	f, err := os.Open(path)
	hasher := sha256.New()
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		log.Fatal(err)
	}
	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash)
}
