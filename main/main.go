package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"

	libXray "github.com/xtls/libxray"
	"github.com/xtls/libxray/nodep"
)

func checkDir(dir string) error {
	if _, err := os.Stat(dir); err == nil {
		err = os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	if err := os.Mkdir(dir, os.ModePerm); err != nil {
		return err
	}
	return nil
}

func downloadFile(url string, writePath string) error {
	client := http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = nodep.WriteBytes(body, writePath)
	return err
}

func saveTimestamp(datDir string) error {
	ts := time.Now().Unix()
	tsText := strconv.FormatInt(ts, 10)
	tsPath := path.Join(datDir, "timestamp.txt")
	return nodep.WriteText(tsText, tsPath)
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	datDir := path.Join(cwd, "dat")
	checkDir(datDir)

	geositeUrl := "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat"
	geositePath := path.Join(datDir, "geosite.dat")
	err = downloadFile(geositeUrl, geositePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	result := libXray.LoadGeoData(datDir, "geosite", "domain")
	if len(result) != 0 {
		fmt.Println("LoadGeoData ", result)
		os.Exit(1)
	}

	geoipUrl := "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat"
	geoipPath := path.Join(datDir, "geoip.dat")
	err = downloadFile(geoipUrl, geoipPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	result = libXray.LoadGeoData(datDir, "geoip", "ip")
	if len(result) != 0 {
		fmt.Println("load geoip ", result)
		os.Exit(1)
	}

	err = saveTimestamp(datDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
