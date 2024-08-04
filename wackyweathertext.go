package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"time"
	// "github.com/piprate/json_gold/ld"
	// _ "github.com/mattn/go-sqlite3"
)

const (
	SUNNY = iota
	CLOUDY
	RAINY
	THUNDERSTORMS
	TORNADO
	HAIL
	SNOW
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
		Forecast         string `json:"forecast"`
		ForecastHourly   string `json:"forecastHourly"`
		RelativeLocation struct {
			LocationProperties struct {
				City  string `json:"city"`
				State string `json:"state"`
			} `json:"properties"`
		} `json:"relativeLocation"`
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
	PeriodName                 string    `json:"name"`
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
		return "https://api.geocode.city/autocomplete?q=" + cityEncoded
	}

	geocodeCityUrl := GeocodeCityUrl(city)
	requestHeaders := make(map[string]string)
	requestHeaders["accept"] = "application/json;charset=utf-8"

	resp, err := GetRequest(geocodeCityUrl, requestHeaders)

	if err != nil {
		return ERRORLONGITUDE, ERRORLATITUDE, err
	}

	// there is a chance that our response empty (especially if a user tries to search for a city that does not exist),
	// if we try to Close() having an empty body it would dereference a nullptr possibly leading to unexpected behavior,
	// ＞﹏＜
	if resp.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(resp.Body)
	}

	err = CheckHttpStatusCode(resp, 200)
	if err != nil {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	//var geocodeResponse []GeocodeResponse
	//err = DecodeJsonResponse(resp, &geocodeResponse)
	//if err != nil {
	//	return ERRORLONGITUDE, ERRORLATITUDE, err
	//}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	// debug
	//fmt.Printf(string(body))

	var rawResponse interface{}
	err = json.Unmarshal(body, &rawResponse)
	if err != nil {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	if reflect.ValueOf(rawResponse).Kind() == reflect.Slice && reflect.ValueOf(rawResponse).Len() == 0 {
		return ERRORLONGITUDE, ERRORLONGITUDE, errors.New("the requested city could not be found")
	}

	var geocodeResponse []GeocodeResponse
	err = json.Unmarshal(body, &geocodeResponse)
	if err != nil {
		return ERRORLONGITUDE, ERRORLONGITUDE, err
	}

	//fmt.Printf("geocodeResponseLength: %v\n", len(geocodeResponse))

	for i := range geocodeResponse {
		if geocodeResponse[i].CountryCode == "US" {
			return geocodeResponse[i].Latitude, geocodeResponse[i].Longitude, nil
		}
	}

	return ERRORLONGITUDE, ERRORLONGITUDE, errors.New("the requested city could not be found")
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

func GetForecastLink(latitude float64, longitude float64) (string, error) {
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

	PrintForecastCityState(metadata)

	return metadata.Properties.Forecast, nil
}

func PrintForecastCityState(metadata LocationMetadata) {
	location := metadata.Properties.RelativeLocation.LocationProperties
	fmt.Printf("%s, %s\n", location.City, location.State)
}

func GetDailyForecasts(link string) ([]ForecastPeriods, error) {
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

func ExtractForecastKeywords(shortForecast string) (int, error) {
	lowerShortForecast := strings.ToLower(shortForecast)

	sunnyStrings := [2]string{"sun", "sunny"}
	cloudyStrings := [2]string{"clouds", "cloudy"}
	rainyStrings := [2]string{"rainy", "rain"}
	thunderstormStrings := [3]string{"thunder", "thunderstorms", "lightning"}
	tornadoString := "tornado"
	hailString := "hail"
	snowStrings := [2]string{"snow", "snowy"}

	if strings.Contains(lowerShortForecast, tornadoString) {
		return TORNADO, nil
	}

	if strings.Contains(lowerShortForecast, hailString) {
		return HAIL, nil
	}

	for _, word := range snowStrings {
		if strings.Contains(lowerShortForecast, word) {
			return SNOW, nil
		}
	}

	for _, word := range thunderstormStrings {
		if strings.Contains(lowerShortForecast, word) {
			return THUNDERSTORMS, nil
		}
	}

	for _, word := range rainyStrings {
		if strings.Contains(lowerShortForecast, word) {
			return RAINY, nil
		}
	}

	for _, word := range cloudyStrings {
		if strings.Contains(lowerShortForecast, word) {
			return CLOUDY, nil
		}
	}

	for _, word := range sunnyStrings {
		if strings.Contains(lowerShortForecast, word) {
			return SUNNY, nil
		}
	}

	return -1, errors.New("could not extract forecast")
}

func renderAscii(forecast ForecastPeriods) (string, error) {
	weatherType, err := ExtractForecastKeywords(forecast.ShortForecast)
	if err != nil {
		return "", err
	}

	if weatherType == TORNADO {
		tornado := "              . '@(@@@@@@@)@. (@@) `  .   '\n     .  @@'((@@@@@@@@@@@)@@@@@)@@@@@@@)@ \n     @@(@@@@@@@@@@))@@@@@@@@@@@@@@@@)@@` .\n  @.((@@@@@@@)(@@@@@@@@@@@@@@))@\\@@@@@@@@@)@@@  .\n (@@@@@@@@@@@@@@@@@@)@@@@@@@@@@@\\\\@@)@@@@@@@@)\n(@@@@@@@@)@@@@@@@@@@@@@(@@@@@@@@//@@@@@@@@@) ` \n .@(@@@@)##&&&&&(@@@@@@@@)::_=(@\\\\@@@@)@@ .   .'\n   @@`(@@)###&&&&&!!;;;;;;::-_=@@\\\\@)@`@.\n   `   @@(@###&&&&!!;;;;;::-=_=@.@\\\\@@     '\n      `  @.#####&&&!!;;;::=-_= .@  \\\\\n            ####&&&!!;;::=_-        `\n             ###&&!!;;:-_=\n              ##&&!;::_=\n             ##&&!;:=\n            ##&&!:-\n           #&!;:-\n          #&!;=\n          #&!-\n           #&=\n   jgs      #&-\n            \\\\#/'"
		return tornado, nil
	}

	if weatherType == HAIL {
		return "", nil
	}

	if weatherType == SNOW {
		return "", nil
	}

	if weatherType == THUNDERSTORMS {
		return "", nil
	}

	if weatherType == RAINY {
		rainCloud := `            ------               _____
           /      \ ___\     ___/    ___
        --/-  ___  /    \/  /  /    /   \
       /     /           \__     //_     \
      /                     \   / ___     |
      |           ___       \/+--/        /
       \__           \       \           /
          \__                 |          /
         \     /____      /  /       |   /
          _____/         ___       \/  /\
               \__      /      /    |    |
             /    \____/   \       /   //
         // / / // / /\    /-_-/\//-__-
          /  /  // /   \__// / / /  //
         //   / /   //   /  // / // /
          /// // / /   /  //  / //
       //   //       //  /  // / /
         / / / / /     /  /    /
      ///  / / /  //  // /  // //
         ///    /    /    / / / /
    ///  /    // / /  // / / /  /
       // ///   /      /// / /`
		return rainCloud, nil
	}

	if weatherType == CLOUDY {
		clouds := "\n                _                                  \n              (`  ).                   _           \n             (     ).              .:(`  )`.       \n)           _(       '`.          :(   .    )      \n        .=(`(      .   )     .--  `.  (    ) )      \n       ((    (..__.:'-'   .+(   )   ` _`  ) )                 \n`.     `(       ) )       (   .  )     (   )  ._   \n  )      ` __.:'   )     (   (   ))     `-'.-(`  ) \n)  )  ( )       --'       `- __.'         :(      )) \n.-'  (_.'          .')                    `(    )  ))\n                  (_  )                     ` __.:'          \n                                        \n"
		return clouds, nil
	}

	if weatherType == SUNNY {
		sun := "      ;   :   ;\n   .   \\_,!,_/   ,\n    `.,'     `.,'\n     /         \\\n~ -- :         : -- ~\n     \\         /\n    ,'`._   _.'`.\n   '   / `!` \\   `\n      ;   :   ;  hjw"
		return sun, nil
	}
	return "", errors.New("could not render forecast")
}

func CheckArgs(args []string) bool {
	if len(args) <= 0 {
		fmt.Println("Usage: 'go build wackyweathertext.go'\n" +
			"'./wackyweathertext cityName'")
		//"If your city is more than one word, make sure to wrap your city name with quotation marks."

		return false
	}
	return true
}

// ฅ^•ﻌ•^ฅ

func main() {
	args := os.Args[1:]
	if !CheckArgs(args) {
		return
	}

	userInput := strings.Join(os.Args[1:], " ")
	lat, long, err := GeocodeCity(userInput)
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

	fmt.Printf("%s\n%d°%s\n%s\n", currForecast.PeriodName, currForecast.Temperature, currForecast.TemperatureUnit, currForecast.DetailedForecast)
	ascii, err := renderAscii(currForecast)
	fmt.Println(ascii)
}
