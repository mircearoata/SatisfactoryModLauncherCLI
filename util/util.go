package util

import (
	"archive/zip"
	"io/ioutil"
	"log"
)

func Check(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func ReadAllFromZip(file *zip.File) []byte {
	fc, err := file.Open()
	Check(err)
	defer fc.Close()

	content, err := ioutil.ReadAll(fc)
	Check(err)

	return content
}
