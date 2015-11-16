package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"net/http"
)

const (
	WhiteHouseLat  float64 = 38.897939
	WhiteHouseLong float64 = -77.036541
	USCapitolLat   float64 = 38.890152
	USCapitolLong  float64 = -77.009096
)

const (
	// Uber API endpoint
	APIUrl string = "https://sandbox-api.uber.com/v1/%s%s"
)

// Getter defines the behavior for all HTTP Get requests
type Getter interface {
	get(c *Client) error
}

// OAuth parameters
type RequestOptions struct {
	ServerToken    string
}

// Client contains the required OAuth tokens and urls and manages
// the connection to the API. All requests are made via this type
type Client struct {
	Options *RequestOptions
}

// Create returns a new API client
func Create(options *RequestOptions) *Client {
	return &Client{options}
}

// Get formulates an HTTP GET request based on the Uber endpoint type
func (c *Client) Get(getter Getter) error {
	if e := getter.get(c); e != nil {
		return e
	}

	return nil
}

// List of time estimates
type TimeEstimates struct {
	StartLatitude  float64
	StartLongitude float64
	Times          []TimeEstimate `json:"times"`
}

// Uber time estimate
type TimeEstimate struct {
	ProductId   string `json:"product_id"`
	DisplayName string `json:"display_name"`
	Estimate    int    `json:"estimate"`
}

func convertToMins(estimate int) int {
	return estimate / 60
}

// Internal method that implements the Getter interface
func (te *TimeEstimates) get(c *Client) error {
	timeEstimateParams := map[string]string{
		"start_latitude":  strconv.FormatFloat(te.StartLatitude, 'f', 2, 32),
		"start_longitude": strconv.FormatFloat(te.StartLongitude, 'f', 2, 32),
	}

	data := c.getRequest("estimates/time", timeEstimateParams)
	if e := json.Unmarshal(data, &te); e != nil {
		return e
	}

	return nil
}

type Products struct {
	Latitude  float64
	Longitude float64
	Products  []Product `json:"products"`
}

// Uber product
type Product struct {
	ProductId   string `json:"product_id"`
	Description string `json:"description"`
	DisplayName string `json:"display_name"`
	Capacity    int    `json:"capacity"`
	Image       string `json:"image"`
}

// Internal method that implements the getter interface
func (pl *Products) get(c *Client) error {
	productParams := map[string]string{
		"latitude":  strconv.FormatFloat(pl.Latitude, 'f', 2, 32),
		"longitude": strconv.FormatFloat(pl.Longitude, 'f', 2, 32),
	}

	data := c.getRequest("products", productParams)
	if e := json.Unmarshal(data, &pl); e != nil {
		return e
	}
	return nil
}


// List of price estimates
type PriceEstimates struct {
	StartLatitude  float64
	StartLongitude float64
	EndLatitude    float64
	EndLongitude   float64
	Prices         []PriceEstimate `json:"prices"`
}

// Uber price estimate
type PriceEstimate struct {
	ProductId       string  `json:"product_id"`
	CurrencyCode    string  `json:"currency_code"`
	DisplayName     string  `json:"display_name"`
	Estimate        string  `json:"estimate"`
	LowEstimate     int     `json:"low_estimate"`
	HighEstimate    int     `json:"high_estimate"`
	SurgeMultiplier float64 `json:"surge_multiplier"`
	Duration        int     `json:"duration"`
	Distance        float64 `json:"distance"`
}

// Internal method that implements the Getter interface
func (pe *PriceEstimates) get(c *Client) error {
	priceEstimateParams := map[string]string{
		"start_latitude":  strconv.FormatFloat(pe.StartLatitude, 'f', 2, 32),
		"start_longitude": strconv.FormatFloat(pe.StartLongitude, 'f', 2, 32),
		"end_latitude":    strconv.FormatFloat(pe.EndLatitude, 'f', 2, 32),
		"end_longitude":   strconv.FormatFloat(pe.EndLongitude, 'f', 2, 32),
	}

	data := c.getRequest("estimates/price", priceEstimateParams)
	if e := json.Unmarshal(data, &pe); e != nil {
		return e
	}
	return nil
}
// Send HTTP request to Uber API
func (c *Client) getRequest(endpoint string, params map[string]string) []byte {
	urlParams := "?"
	params["server_token"] = c.Options.ServerToken
	for k, v := range params {
		if len(urlParams) > 1 {
			urlParams += "&"
		}
		urlParams += fmt.Sprintf("%s=%s", k, v)
	}

	url := fmt.Sprintf(APIUrl, endpoint, urlParams)

	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	return data
}

func main() {
	// Read API auth options
	var options RequestOptions
	options.ServerToken = "VlGEE0x4Vy_xQ1-LMobj-4i6_xcv1Uo-mIlRNefb"
	client := Create(&options)

	// Retrieve products based on lat/long coords
	pl := &Products{}
	pl.Latitude = WhiteHouseLat
	pl.Longitude = WhiteHouseLong
	if e := client.Get(pl); e != nil {
		log.Fatal(e)
	}

	fmt.Println("Here are the Uber options available for your area: \n")
	for _, product := range pl.Products {
		if product.ProductId =="dee8691c-8b48-4637-b048-300eee72d58d"{
			fmt.Println(product.DisplayName + ": " + product.Description)
		}
	}

	// Retrieve price estimates based on start and end lat/long coords
	pe := &PriceEstimates{}
	pe.StartLatitude = WhiteHouseLat
	pe.StartLongitude = WhiteHouseLong
	pe.EndLatitude = USCapitolLat
	pe.EndLongitude = USCapitolLong
	if e := client.Get(pe); e != nil {
		log.Fatal(e)
	}

	fmt.Println("\nHere are the Uber price estimates from The White House to the United States Capitol: \n")
	for _, price := range pe.Prices {
		if price.ProductId =="dee8691c-8b48-4637-b048-300eee72d58d"{
		fmt.Println(price.DisplayName + ": " + price.Estimate + "; Surge: " + strconv.FormatFloat(price.SurgeMultiplier, 'f', 2, 32))
		}
	}

	// Retrieve ETA estimates based on start lat/long coords
	te:= &TimeEstimates{}
	te.StartLatitude = WhiteHouseLat
	te.StartLongitude = WhiteHouseLong
	if e := client.Get(te); e != nil {
		log.Fatal(e)
	}

	fmt.Println("\nHere are the Uber ETA estimates if leaving from The White House: \n")
	for n, eta := range te.Times {
		if eta.ProductId =="dee8691c-8b48-4637-b048-300eee72d58d"{
		fmt.Println(eta.DisplayName + ": " + strconv.Itoa(eta.Estimate/60))
		}
		if n == len(te.Times)-1 {
			fmt.Print("\n")
		}
	}
}