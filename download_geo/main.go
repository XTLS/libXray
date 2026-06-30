package main

import (
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

type invokeResponseEnvelope struct {
	Success bool   `json:"success"`
	Err     string `json:"error,omitempty"`
}

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

func saveTimestamp(datDir string, fileName string) error {
	ts := time.Now().Unix()
	tsText := strconv.FormatInt(ts, 10)
	tsPath := path.Join(datDir, fileName)
	return nodep.WriteText(tsText, tsPath)
}

func parseInvokeResponse(text string) (invokeResponseEnvelope, error) {
	var response invokeResponseEnvelope
	err := json.Unmarshal([]byte(text), &response)
	return response, err
}

func makeLoadGeoDataRequest(datDir string, name string, geoType string) (string, error) {
	payload, err := json.Marshal(&libXray.CountGeoDataRequest{
		Name:    name,
		GeoType: geoType,
	})
	if err != nil {
		return "", err
	}
	request := libXray.LibXrayInvokeRequest{
		APIVersion: 1,
		Method:     libXray.LibXrayMethodCountGeoData,
		Env: &libXray.LibXrayEnvJson{
			AssetLocation: datDir,
			CertLocation:  datDir,
		},
		Payload: payload,
	}
	data, err := json.Marshal(&request)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func downloadDat(url string, datDir string, fileName string, geoType string) {
	datFile := fmt.Sprintf("%s.dat", fileName)
	geositePath := path.Join(datDir, datFile)
	err := downloadFileIfNotExists(url, geositePath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	geoReq, err := makeLoadGeoDataRequest(datDir, fileName, geoType)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	res := libXray.Invoke(geoReq)
	resp, err := parseInvokeResponse(res)
	if err != nil || !resp.Success {
		fmt.Println("Failed to load geosite:", url, res, resp.Err)
		os.Exit(1)
	}
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
	downloadDat("https://github.com/v2fly/domain-list-community/releases/latest/download/dlc.dat", datDir, "geosite", "domain")
	// Download geoip.dat
	downloadDat("https://github.com/v2fly/geoip/releases/latest/download/geoip.dat", datDir, "geoip", "ip")

	// Save timestamp
	err = saveTimestamp(datDir, "timestamp.txt")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("Geo data setup completed successfully.")
}
