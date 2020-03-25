package main

import (
	"archive/zip"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitly/go-simplejson"
	uuid "github.com/satori/go.uuid"
)

var kaiosAppPath = flag.String("path", "app", "KaiOS app path")
var verboseFlag = flag.Bool("verbose", false, "Verbose output")
var outputPath = flag.String("output", "app.zip", "OmniSD package path")

func main() {
	flag.Parse()
	fmt.Println("KaiPack by zjyl1994")
	if *verboseFlag {
		fmt.Println("https://github.com/zjyl1994/kaipack\n=============")
	}
	if *verboseFlag {
		fmt.Println(">> packing app in zip.")
	}
	packagedAppZip, err := zipToMem(*kaiosAppPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if *verboseFlag {
		fmt.Println("ZIP_LENGTH::", len(packagedAppZip))
		fmt.Println(">> zip pack success.")
	}
	bMetadata, err := genMetadata(*kaiosAppPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if *verboseFlag {
		fmt.Println(string(bMetadata))
		fmt.Println(">> metadata generated.")
	}
	err = packSoftware(*outputPath, bMetadata, packagedAppZip)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if *verboseFlag {
		fmt.Println(">> package generated.")
	}
	fmt.Println("All done.")
}

func packSoftware(dst string, metadata, appZip []byte) error {
	zipfile, err := os.Create(*outputPath)
	if err != nil {
		return err
	}
	defer zipfile.Close()
	archive := zip.NewWriter(zipfile)
	defer archive.Close()
	metaWriter, err := archive.Create("metadata.json")
	if err != nil {
		return err
	}
	_, err = metaWriter.Write(metadata)
	if err != nil {
		return err
	}
	bodyWriter, err := archive.Create("application.zip")
	if err != nil {
		return err
	}
	_, err = bodyWriter.Write(appZip)
	return err
}

func genMetadata(source string) ([]byte, error) {
	webappFile := filepath.Join(source, "manifest.webapp")
	bJson, err := ioutil.ReadFile(webappFile)
	if err != nil {
		return nil, err
	}
	json, err := simplejson.NewJson(bJson)
	if err != nil {
		return nil, err
	}
	var manifestURL string
	if origin, ok := json.CheckGet("origin"); ok {
		originPath := origin.MustString()
		if strings.HasSuffix(originPath, "/") {
			manifestURL = originPath + "manifest.webapp"
		} else {
			manifestURL = originPath + "/manifest.webapp"
		}
	} else {
		manifestURL = fmt.Sprintf("app://%s/manifest.webapp", uuid.NewV4().String())
	}
	jsonObj := simplejson.New()
	jsonObj.Set("version", 1)
	jsonObj.Set("manifestURL", manifestURL)
	return jsonObj.MarshalJSON()
}

func zipToMem(source string) (data []byte, err error) {
	buf := new(bytes.Buffer)
	archive := zip.NewWriter(buf)
	source, err = filepath.Abs(source)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(source)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("source not dir")
	}
	filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = strings.TrimPrefix(path, source)
		if info.IsDir() && header.Name == "" {
			return nil
		}
		header.Name = filepath.ToSlash(header.Name)
		if info.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}
		header.Name = strings.TrimPrefix(header.Name, "/")
		if *verboseFlag {
			fmt.Println(header.Name)
		}
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		_, err = io.Copy(writer, file)
		return err
	})
	archive.Close()
	return buf.Bytes(), nil
}
