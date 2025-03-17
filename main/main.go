package main

import (
	"encoding/base64"
	"encoding/json"
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

func ensureDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

func downloadFileIfNotExists(url string, writePath string) error {
	if _, err := os.Stat(writePath); err == nil {
		fmt.Printf("File already exists: %s, skipping download.\n", writePath)
		return nil
	}

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

func parseCallResponse(text string) (nodep.CallResponse[string], error) {
	var response nodep.CallResponse[string]
	decoded, err := base64.StdEncoding.DecodeString(text)
	if err != nil {
		return response, err
	}
	err = json.Unmarshal(decoded, &response)
	return response, err
}

func makeLoadGeoDataRequest(datDir string, name string, geoType string) (string, error) {
	var request libXray.CountGeoDataRequest
	request.DatDir = datDir
	request.Name = name
	request.GeoType = geoType

	data, err := json.Marshal(&request)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	datDir := path.Join(cwd, "dat")
	err = ensureDir(datDir)
	if err != nil {
		fmt.Println("Failed to ensure directory:", err)
		os.Exit(1)
	}

	// Download geosite.dat
	geositeUrl := "https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat"
	geositePath := path.Join(datDir, "geosite.dat")
	err = downloadFileIfNotExists(geositeUrl, geositePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Load geosite
	geoSiteReq, err := makeLoadGeoDataRequest(datDir, "geosite", "domain")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res := libXray.CountGeoData(geoSiteReq)
	resp, err := parseCallResponse(res)
	if err != nil || !resp.Success {
		fmt.Println("Failed to load geosite:", res)
		os.Exit(1)
	}

	// Download geoip.dat
	geoipUrl := "https://github.com/v2fly/geoip/releases/latest/download/geoip.dat"
	geoipPath := path.Join(datDir, "geoip.dat")
	err = downloadFileIfNotExists(geoipUrl, geoipPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Load geoip
	geoIpReq, err := makeLoadGeoDataRequest(datDir, "geoip", "ip")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res = libXray.CountGeoData(geoIpReq)
	resp, err = parseCallResponse(res)
	if err != nil || !resp.Success {
		fmt.Println("Failed to load geoip:", res)
		os.Exit(1)
	}

	// Save timestamp
	err = saveTimestamp(datDir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Geo data setup completed successfully.")
}
