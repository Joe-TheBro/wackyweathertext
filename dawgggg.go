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

func geocodeCity(city string) (float64, float64, error) {
	client := &http.Client{}
	cityEncoded := url.QueryEscape(city)
	req, err := http.NewRequest("GET", "https://api.geocode.city/autocomplete?limit=1&q="+cityEncoded, nil)
	if err != nil {
		return 0, 0, err
	}

	req.Header.Set("accept", "application/json;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	dec := json.NewDecoder(resp.Body)
	var formattedResponse []geocodeResponse
	if err := dec.Decode(&formattedResponse); err != nil {
		if err == io.EOF {
			return 0, 0, errors.New("no valid geocode response found")
		}
		return 0, 0, err
	}

	if len(formattedResponse) == 0 {
		return 0, 0, errors.New("no results found")
	}

	result := formattedResponse[0]
	fmt.Printf("%s\n%f\n%f\n%s\n", result.Name, result.Longitude, result.Latitude, result.Country)
	return result.Latitude, result.Longitude, nil
}

func main() {
	lat, long, err := geocodeCity("Sioux Falls")
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
