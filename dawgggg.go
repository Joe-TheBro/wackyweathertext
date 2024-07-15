package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	// "github.com/piprate/json_gold/ld"
	// _ "github.com/mattn/go-sqlite3"
)

const ERRORLATITUDE float64 = -91.0
const ERRORLONGITUDE float64 = -181.0

type geocodeResponse struct {
	Name        string  `json:"name"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	District    string  `json:"district"`
	Timezone    string  `json:"timezone"`
	Population  int     `json:"population"`
}

func GeocodeCity(city string) (float64, float64, error) {
	url := GeocodeCityUrl(city)
	header := "application/json;charset=utf-8"

	resp, err := GetRequest(url, header)
	defer resp.Body.Close()

	if IsError(err) {
		return ERRORLONGITUDE, ERRORLATITUDE, err
	}

	err = Check200StatusCode(resp)
	if IsError(err) {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	result, err := DecodeResponse(resp)
	if IsError(err) {
		return ERRORLONGITUDE, ERRORLATITUDE, err
	}

	fmt.Printf("%s\n%f\n%f\n%s\n", result.Name, result.Longitude, result.Latitude, result.Country)
	return result.Latitude, result.Longitude, nil
}

func GeocodeCityUrl(city string) string {
	cityEncoded := url.QueryEscape(city)
	return "https://api.geocode.city/autocomplete?limit=1&q=" + cityEncoded
}

func GetRequest(url string, header string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if IsError(err) {
		return nil, err
	}
	req.Header.Set("accept", header)
	resp, err := client.Do(req)

	if IsError(err) {
		return nil, err
	}

	return resp, nil
}

func Check200StatusCode(resp *http.Response) error {
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return err
	}
	return nil
}

func DecodeResponse(resp *http.Response) (*geocodeResponse, error) {
	dec := json.NewDecoder(resp.Body)

	var formattedResponse []geocodeResponse
	if err := dec.Decode(&formattedResponse); err != nil {
		if err == io.EOF {
			return nil, errors.New("no valid geocode response found")
		}
		return nil, err
	}

	if len(formattedResponse) == 0 {
		return nil, errors.New("no results found")
	}

	coords := formattedResponse[0]
	return &coords, nil
}

func IsError(err error) bool {
	return err != nil
}

// ฅ^•ﻌ•^ฅ

func main() {
	lat, long, err := GeocodeCity("Sioux Falls")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("lat: %f, long: %f\n", lat, long)
}

/*type Weather interface {
	renderAsciiArt() string
}

type Sunny struct {
1
}*/
