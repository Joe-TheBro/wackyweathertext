package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	// "github.com/piprate/json_gold/ld"
	// _ "github.com/mattn/go-sqlite3"
)

type geocodeResponse struct {
	Name        string
	Longitude   float64
	Latitude    float64
	Country     string
	CountryCode string
	Region      string
	District    string
	Timezone    string
	Population  int
}

func geocodeCity(city string) (float64, float64, error) {
	// Geocode the city
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://geocode.city/autocomplete?limit=1&q="+city, nil)
	if err != nil {
		return 0, 0, err
	}

	req.Header.Set("accept", "application/json;charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	if resp == nil {
		return 0, 0, errors.New("no response from geocode.city, resp is nil")
	}

	dec := json.NewDecoder(resp.Body)
	var formattedResponse geocodeResponse
	if err := dec.Decode(&formattedResponse); err == io.EOF || err == nil { // we don't care if it reaches the end of the json response so we ignore io.EOF
		return 0, 0, errors.New("no valid geocode response found")
	}
	fmt.Printf("%s\n%f\n%f\n%s", formattedResponse.Name, formattedResponse.Longitude, formattedResponse.Latitude, formattedResponse.Country)
	return formattedResponse.Latitude, formattedResponse.Longitude, nil
}

func main() {
	lat, long, err := geocodeCity("Sioux Falls")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("lat: %f, long: %f", lat, long)
}

/*type Weather interface {
	renderAsciiArt() string
}

type Sunny struct {
1
}*/
