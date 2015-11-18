
package main


import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"net/http"
	"github.com/julienschmidt/httprouter"
	"gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "os"
    "math"
)

const (
	// Uber API endpoint
	APIUrl string = "https://sandbox-api.uber.com/v1/%s%s"
)

type Input_locations struct {

Start_loc string		`json:"starting_from_location_id"`
Remaining_loc []string  `json:"location_ids"`

}

type Post_response struct{

Id_post			bson.ObjectId    `json:"id" bson:"_id"`
Status 			string 			`json:"status" bson:"status"`		
Start_loc       string          `json:"starting_from_location_id" bson:"starting_from_location_id"`
Best_route		[]string        `json:"best_route_location_ids" bson:"best_route_location_ids"`
Total_uber_cost int 			`json:"total_uber_costs" bson:"total_uber_costs"`
Total_uber_duration int 		`json:"total_uber_duration" bson:"total_uber_duration"`
Total_distance	float64			`json:"total_distance" bson:"total_distance"`

}

type Uber_api_response struct {

	Uber_distance 	float64
	Uber_duration 	int
	Uber_cost		int
}

type (  
    UserResponse struct {
        Id      bson.ObjectId          `json:"id" bson:"_id"`
        Name    string       `json:"name" bson:"name"`
        Address string       `json:"address" bson:"address"`
        City    string       `json:"city" bson:"city"`
        State   string       `json:"state" bson:"state"`
        Zip     string       `json:"zip" bson:"zip"`
        Cc      Coordinate   `json:"coordinate" bson:"coordinate"`
    }
)

type Coordinate struct{
        Lat     float64      `json:"lat" bson:"lat"`
        Lng     float64      `json:"lng" bson:"lng"`
}

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

	mux := httprouter.New()
    mux.POST("/trips",post)
    mux.GET("/trips/:id",get)
    server := http.Server{
            Addr:        "0.0.0.0:8080",
            Handler: mux,
    }
    server.ListenAndServe()
	
}

func get(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
    log.Println("inside Get")
    identifier := p.ByName("id")
    //log.Println("id is : ", identifier)
    if !bson.IsObjectIdHex(identifier) {
        rw.WriteHeader(404)
        return
    }
    oid := bson.ObjectIdHex(identifier)
    log.Println("id is : ", oid)
    sess, err := mgo.Dial("mongodb://Richa:Indore#1@ds041934.mongolab.com:41934/assignment_2_db")
    if err != nil {
       fmt.Printf("Can't connect to mongo, go error %v\n", err)
       os.Exit(1)
    }
    defer sess.Close()
    sess.SetSafe(&mgo.Safe{})
    collection := sess.DB("assignment_2_db").C("routes")
    var get_response Post_response
    err = collection.Find(bson.M{"_id":oid}).One(&get_response)
    if err != nil {
    // handle error
       fmt.Fprintf(rw,"Record not found",err)
       return
    }  
    uj, _ := json.Marshal(get_response)
    rw.Header().Set("Content-Type", "application/json")
    rw.WriteHeader(200)
    fmt.Fprintf(rw, "%s", uj)
}

func post(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
    log.Println("inside post")
    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        http.Error(rw, err.Error(), http.StatusInternalServerError)
        return
    }
   var u Input_locations
   err = json.Unmarshal(body, &u)
    if (err != nil ) {
        http.Error(rw, "Bad Request, check request payload", http.StatusBadRequest)
        return
    }
    var rest_loc []string
    start := u.Start_loc
	rest_loc = u.Remaining_loc
	i:= len(u.Remaining_loc)-1
	var temp = make([]Uber_api_response,i+1)
	var route = make ([]string,i+1)
	var j int
	total_cost := 0
	total_dist := 0.0
	total_duration := 0
	for i!=-1 {
		for j=0; j<=i; j++ {
		    temp[j] = uber_api(start, rest_loc[j])

		}
	min := temp[0].Uber_cost
	index :=0
	var k int
	for k = 0; k < len(temp); k++{
			if temp[k].Uber_cost<min {
			min = temp[k].Uber_cost
			index = k
		}

	}
	start = rest_loc[index]
	route[len(u.Remaining_loc)-(i+1)]=start
	j:=0
	for k =0;k<len(rest_loc);k++{
		if (k!=index){
			rest_loc[j] = rest_loc[k]
			j=j+1
		}
	}
	total_cost = total_cost + temp[index].Uber_cost
	total_dist = total_dist + temp[index].Uber_distance
	total_duration = total_duration + temp[index].Uber_duration
	length := i
	temp = make([]Uber_api_response,length)
    i=i-1

}
	var final_response Post_response
	final_response.Id_post = bson.NewObjectId()
	final_response.Total_uber_cost = total_cost
	final_response.Total_uber_duration = total_duration
	final_response.Total_distance = math.Ceil(total_dist)
	final_response.Start_loc = u.Start_loc
	final_response.Best_route = route
	final_response.Status = "Planning"
	Insert_to_mongodb(final_response)
    uj, _ := json.Marshal(final_response)
  	rw.Header().Set("Content-Type", "application/json")
  	rw.WriteHeader(201)
  	fmt.Fprintf(rw, "%s", uj)
}

func uber_api(Start_loc string,Remaining_loc string) Uber_api_response{

	oid1:=bson.ObjectIdHex(Start_loc)
	oid2:=bson.ObjectIdHex(Remaining_loc)
    sess, err := mgo.Dial("mongodb://Richa:Indore#1@ds041934.mongolab.com:41934/assignment_2_db")
    if err != nil {
       fmt.Printf("Can't connect to mongo, go error %v\n", err)
       os.Exit(1)
    }
    defer sess.Close()
    sess.SetSafe(&mgo.Safe{})
    collection := sess.DB("assignment_2_db").C("loc")
    var user UserResponse
    err = collection.Find(bson.M{"_id":oid1}).One(&user)
    if err != nil {
    	log.Println("Record not found",err)
       
    }  
    var options RequestOptions
	options.ServerToken = "VlGEE0x4Vy_xQ1-LMobj-4i6_xcv1Uo-mIlRNefb"
	client := Create(&options)

    pl := &Products{}
	pl.Latitude = user.Cc.Lat
	pl.Longitude = user.Cc.Lng
	if e := client.Get(pl); e != nil {
		log.Fatal(e)
	}
	i:=0
	var productid string
	for _, product := range pl.Products {
		if(i == 0){
			productid = product.ProductId
		}
		i=i+1

	}

    pe := &PriceEstimates{}
	pe.StartLatitude = user.Cc.Lat 
	pe.StartLongitude = user.Cc.Lng
	err = collection.Find(bson.M{"_id":oid2}).One(&user)
    if err != nil {
    	log.Println("Record not found",err)
    }  
	pe.EndLatitude = user.Cc.Lat
	pe.EndLongitude = user.Cc.Lng	
 
	if e := client.Get(pe); e != nil {
		log.Fatal(e)
	}
	//var price_estimate int
	var uber_response Uber_api_response
	for _, price := range pe.Prices {
		if price.ProductId == productid{
		uber_response.Uber_cost = price.LowEstimate
		uber_response.Uber_duration= price.Duration
		uber_response.Uber_distance = price.Distance
		}
	}

	// Retrieve ETA estimates based on start lat/long coords
	te:= &TimeEstimates{}
	te.StartLatitude = pe.StartLatitude
	te.StartLongitude = pe.StartLongitude
	if e := client.Get(te); e != nil {
		log.Fatal(e)
	}

	for n, eta := range te.Times {
		if eta.ProductId ==productid{
		//fmt.Println(eta.DisplayName + ": " + strconv.Itoa(eta.Estimate/60))
		}
		if n == len(te.Times)-1 {
			fmt.Print("\n")
		}
	}
	return uber_response
}


func Insert_to_mongodb(final_response Post_response){
  sess, err := mgo.Dial("mongodb://Richa:Indore#1@ds041934.mongolab.com:41934/assignment_2_db")
  if err != nil {
    fmt.Printf("Can't connect to mongo, go error %v\n", err)
    os.Exit(1)
  }
  defer sess.Close()
  sess.SetSafe(&mgo.Safe{})
  collection := sess.DB("assignment_2_db").C("routes")
  err = collection.Insert(final_response)
  if (err != nil ) {
  		log.Println("error in inserting to database",err)
        
    }

}
