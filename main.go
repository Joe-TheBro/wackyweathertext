package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"reflect"
	"time"
	// "github.com/piprate/json_gold/ld"
	// _ "github.com/mattn/go-sqlite3"
)

type GeocodeResponse struct {
	Name        string  `json:"name"`
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	District    string  `json:"district"`
	Timezone    string  `json:"timezone"`
	Population  int     `json:"population"`
}

type LocationMetadata struct {
	Properties struct {
		Forecast       string `json:"forecast"`
		ForecastHourly string `json:"forecastHourly"`
	} `json:"properties"`
}

type ForecastInfo struct {
	Properties struct {
		GeneratedAt time.Time         `json:"generatedAt"`
		Periods     []ForecastPeriods `json:"periods"`
	} `json:"properties"`
}

type ForecastPeriods struct {
	Number                     int       `json:"number"`
	Name                       string    `json:"name"`
	StartTime                  time.Time `json:"startTime"`
	EndTime                    time.Time `json:"endTime"`
	IsDayTime                  bool      `json:"isDayTime"`
	Temperature                int       `json:"temperature"`
	TemperatureUnit            string    `json:"temperatureUnit"`
	TemperatureTrend           string    `json:"temperatureTrend"`
	ProbabilityOfPrecipitation struct {
		UnitCode string `json:"unitCode"`
		Value    int    `json:"value"`
	} `json:"probabilityOfPrecipitation"`
	WindSpeed        string `json:"windSpeed"`
	WindDirection    string `json:"windDirection"`
	ShortForecast    string `json:"shortForecast"`
	DetailedForecast string `json:"detailedForecast"`
}

func GeocodeCity(city string) (float64, float64, error) {
	const ERRORLATITUDE float64 = -91.0
	const ERRORLONGITUDE float64 = -181.0

	GeocodeCityUrl := func(city string) string {
		cityEncoded := url.QueryEscape(city)
		return "https://api.geocode.city/autocomplete?limit=1&q=" + cityEncoded
	}

	url := GeocodeCityUrl(city)
	requestHeaders := make(map[string]string)
	requestHeaders["accept"] = "application/json;charset=utf-8"

	resp, err := GetRequest(url, requestHeaders)

	// while unlikely, there is a chance that our response empty, if we try to Close()
	// having an empty body it would dereference a nullptr possibly leading to unexpected behavior, ＞﹏＜
	if resp != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		return ERRORLONGITUDE, ERRORLATITUDE, err
	}

	err = CheckHttpStatusCode(resp, 200)
	if err != nil {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	var geocodeResponse []GeocodeResponse
	err = DecodeJsonResponse(resp, &geocodeResponse)
	if err != nil {
		return ERRORLONGITUDE, ERRORLATITUDE, err
	}

	//* Debug print statement
	//fmt.Printf("%s\n%f\n%f\n%s\n", geocodeResponse[0].Name, geocodeResponse[0].Longitude, geocodeResponse[0].Latitude, geocodeResponse[0].Country)
	return geocodeResponse[0].Latitude, geocodeResponse[0].Longitude, nil
}

func GetRequest(url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func CheckHttpStatusCode(resp *http.Response, status int) error {
	if resp.StatusCode != status {
		err := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		return err
	}
	return nil
}

func DecodeJsonResponse(resp *http.Response, v interface{}) error {
	dec := json.NewDecoder(resp.Body)

	// First, decode into a generic interface{} to check the type
	var rawResponse interface{}
	if err := dec.Decode(&rawResponse); err != nil {
		if err == io.EOF {
			return errors.New("no valid response found")
		}
		return err
	}

	rawJSON, err := json.Marshal(rawResponse)
	if err != nil {
		return err
	}

	// Handle if the response is an array or a single object
	valueType := reflect.TypeOf(v).Elem()
	/*fmt.Printf("Expected type: %s\n", valueType.Kind())
	fmt.Printf("rawResponse Type: %s\n", reflect.TypeOf(rawResponse).Kind())*/
	if reflect.TypeOf(rawResponse).Kind() == reflect.Slice {
		// Ensure v is a slice type
		if valueType.Kind() != reflect.Slice {
			return errors.New("expected a slice type for the response")
		}

		// Create a new slice of the appropriate type
		slicePtr := reflect.New(reflect.SliceOf(valueType.Elem())).Interface()
		if err := json.Unmarshal(rawJSON, slicePtr); err != nil {
			return err
		}

		// Set the original pointer to the new slice
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(slicePtr).Elem())
	} else {
		// Ensure v is not a slice type
		if valueType.Kind() == reflect.Slice {
			return errors.New("expected a single object type for the response")
		}

		// Create a new instance of the appropriate type
		objPtr := reflect.New(valueType).Interface()
		if err := json.Unmarshal(rawJSON, objPtr); err != nil {
			return err
		}

		// Set the original pointer to the new object
		reflect.ValueOf(v).Elem().Set(reflect.ValueOf(objPtr).Elem())
	}

	return nil
}

func GetForecastLink(latitude float64, longitude float64) (link string, err error) {
	locationLink := fmt.Sprintf("https://api.weather.gov/points/%.4g,%.4g", latitude, longitude)

	requestHeaders := make(map[string]string)
	requestHeaders["accept"] = "application/geo+json"

	resp, err := GetRequest(locationLink, requestHeaders)
	if err != nil {
		return "", err
	}

	err = CheckHttpStatusCode(resp, 200)
	if err != nil {
		return "", err
	}

	var metadata LocationMetadata
	err = DecodeJsonResponse(resp, &metadata)
	if err != nil {
		return "", err
	}

	return metadata.Properties.Forecast, nil
}

func GetDailyForecasts(link string) (forecasts []ForecastPeriods, err error) {
	requestHeaders := make(map[string]string)
	requestHeaders["accept"] = "application/geo+json"

	resp, err := GetRequest(link, requestHeaders)
	if err != nil {
		return nil, err
	}

	err = CheckHttpStatusCode(resp, 200)
	if err != nil {
		return nil, err
	}

	var forecastInfo ForecastInfo
	err = DecodeJsonResponse(resp, &forecastInfo)
	if err != nil {
		return nil, err
	}

	forecast := forecastInfo.Properties.Periods

	return forecast, nil
}

// ฅ^•ﻌ•^ฅ

func main() {
	lat, long, err := GeocodeCity("Sioux Falls")
	if err != nil {
		log.Fatal(err)
	}

	forecastLink, err := GetForecastLink(lat, long)
	if err != nil {
		log.Fatal(err)
	}

	forecasts, err := GetDailyForecasts(forecastLink)
	if err != nil {
		log.Fatal(err)
	}

	currForecast := forecasts[0]

	fmt.Printf("%s\n%d°%s\n%s", currForecast.Name, currForecast.Temperature, currForecast.TemperatureUnit, currForecast.DetailedForecast)
}
