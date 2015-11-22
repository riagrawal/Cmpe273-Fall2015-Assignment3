
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
    "bytes"
)

const (
	// Uber API endpoint
	APIUrl string = "https://sandbox-api.uber.com/v1/%s%s"
)

type Input_locations struct {

Start_loc string		`json:"starting_from_location_id"`
Remaining_loc []string  `json:"location_ids"`

}


type Trip struct{

Id_put			bson.ObjectId    `json:"id" bson:"_id"`
Status_put 			string 			`json:"status" bson:"status"`		
Start_loc_put      string          `json:"starting_from_location_id" bson:"starting_from_location_id"`
Next_loc_put		string          `json:"next_destination_location_id,omitempty" bson: "next_destination_location_id"`
Best_route_put		[]string        `json:"best_route_location_ids" bson:"best_route_location_ids"`
Total_uber_cost_put int 			`json:"total_uber_costs" bson:"total_uber_costs"`
Total_uber_duration_put int 		`json:"total_uber_duration" bson:"total_uber_duration"`
Total_distance_put	float64			`json:"total_distance" bson:"total_distance"`
Uber_eta_put		int				`json:"uber_wait_time_eta,omitempty" bson :"uber_wait_time_eta"`

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
	//ServerToken    string
	AccessToken	   string
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

type Ride_request struct{
	Pid     	string 		`json:"product_id"`
	StartLat  float64       `json:"start_latitude"`
	StartLng float64        `json:"start_longitude"`
	EndLat    float64        `json:"end_latitude"`
	EndLng   float64         `json:"end_longitude"`

}

type Ride_request_response struct{
	Request_id		string   `json:"request_id"`
   	Status_request	string 	 `json:"status"`
   Vehicle			string 	 `json:"vehicle"`
   Driver			string 	 `json:"driver"`
   Location 		string 	 `json:"location"`
   Eta 				int      `json:"eta"`
   Surge_Multiplier int      `json:"surge_multiplier"`

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
	params["access_token"] = c.Options.AccessToken
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
    mux.PUT("/trips/:id/request",put)
    server := http.Server{
            Addr:        "0.0.0.0:8080",
            Handler: mux,
    }
    server.ListenAndServe()
	
}

func get(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
    log.Println("inside Get")
    identifier := p.ByName("id")
    if !bson.IsObjectIdHex(identifier) {
        rw.WriteHeader(404)
        return
    }
    oid := bson.ObjectIdHex(identifier)
    var get_response Trip
    get_response = find_record(oid)
    get_response.Next_loc_put =""
    get_response.Uber_eta_put = 0
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
	length := 1
	temp = make([]Uber_api_response,length)
	var final_response Trip
	temp[0]= uber_api(rest_loc[len(u.Remaining_loc)-1],u.Start_loc)
	total_cost = total_cost+temp[0].Uber_cost
	final_response.Id_put = bson.NewObjectId()
	final_response.Total_uber_cost_put = total_cost
	final_response.Total_uber_duration_put = total_duration
	final_response.Total_distance_put = math.Ceil(total_dist)
	final_response.Start_loc_put  = u.Start_loc
	final_response.Best_route_put = route
	final_response.Status_put = "Planning"
	Insert_to_mongodb(final_response)
    uj, _ := json.Marshal(final_response)
  	rw.Header().Set("Content-Type", "application/json")
  	rw.WriteHeader(201)
  	fmt.Fprintf(rw, "%s", uj)
}

func put(rw http.ResponseWriter, req *http.Request, p httprouter.Params) {
	log.Println("inside Put!!!!")
    identifier := p.ByName("id")
    if !bson.IsObjectIdHex(identifier) {
        rw.WriteHeader(404)
        return
    }
    oid := bson.ObjectIdHex(identifier)
    var start_location string
    var i int
    var next_location string
 	var trip Trip
 	trip = find_record (oid)
 	if(trip.Start_loc_put == trip.Next_loc_put){
 		fmt.Fprintf(rw,"%s","\"Message\":\"Trip Completed\"")
 		log.Println("Trip Completed")
 		return
 	}
 	trip.Status_put = "Requesting"
 	trip.Uber_eta_put = 5
 	if (trip.Next_loc_put == ""){
 		start_location = trip.Start_loc_put
 		trip.Next_loc_put = trip.Best_route_put[0]
 		next_location = trip.Next_loc_put
 		
 	}else {
 		start_location = trip.Next_loc_put
 		for i =0 ; i< len(trip.Best_route_put); i++{
 			if(start_location == trip.Best_route_put[i]){
 					if(i==len(trip.Best_route_put)-1){
 							next_location = trip.Start_loc_put
 							trip.Next_loc_put = trip.Start_loc_put
 							break
 					}
 					next_location = trip.Best_route_put[i+1]
 					trip.Next_loc_put = trip.Best_route_put[i+1]
 			}
 		}
 	} 
 	trip.Uber_eta_put = ride_request(start_location,next_location)
 	trip =Update_mongodb(trip)
    uj, _ := json.Marshal(trip)
  	rw.Header().Set("Content-Type", "application/json")
  	rw.WriteHeader(201)
  	fmt.Fprintf(rw, "%s", uj)

}

func ride_request(start_point string, next_point string ) int{
	oid1:=bson.ObjectIdHex(start_point)
	oid2:=bson.ObjectIdHex(next_point)
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
	//options.AccessToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSJdLCJzdWIiOiJjMWZjYjc3Yy0yOTAxLTQ2NmYtOWUyMi0xZTExMTZhYWVkMDYiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjYzNzhiMGNjLWZjMTktNDcyMC05ODUxLTJlNmU2YWU1Mjc3OSIsImV4cCI6MTQ1MDUwNzc3NCwiaWF0IjoxNDQ3OTE1NzczLCJ1YWN0IjoiQjNlSzF4ck5IR2lpcG0wbnVJVGo5bGdqazV3eGQxIiwibmJmIjoxNDQ3OTE1NjgzLCJhdWQiOiJrQTMzVkhwSFdWSUlKLWVyMlVVaHZlcDU2S3JuYjBoZiJ9.CWWNX2ElfbqQYJcu_YRsTUiqkSBBlMd6BoOnjroLoYHfnNxFeJI61CU8Bd7qkY2Z_u8WC3lkWJR-3WtJiTDdD-kph-Imw_Nqa3XQalrpsGwCyOvXS57_zWYb8v9E3o3GnRwsDr6TeoEmDkEdikKsK3RlQ6W-U9ywIE3IXXFwaJZyAqS-Q8U_D6NxZb0CQF8212pJSGn34cXEucRtiBJPzuS1VsXLcd6Tgk2AcOXlUU43ukzU_UCK5YBDWvbSXgDISMqBkZHFgxMSubWyTAtdEGtjL-N1Zm9TG-bB66mcPCvAeeg8aam48NX7yGCVasH6z2-U2FCjXSr00rKUcapleQ"
	options.AccessToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3QiLCJoaXN0b3J5Il0sInN1YiI6ImMxZmNiNzdjLTI5MDEtNDY2Zi05ZTIyLTFlMTExNmFhZWQwNiIsImlzcyI6InViZXItdXMxIiwianRpIjoiOTY1ZTFjNjgtOTY5Ni00ZDQ1LWFlMWQtMTk5MjU5NzQ2ZDJiIiwiZXhwIjoxNDUwNTkzNTM0LCJpYXQiOjE0NDgwMDE1MzMsInVhY3QiOiJ3R1JyMjlLVGRFdUNHY2ExcHJtVFZRQmhjWmlEc3UiLCJuYmYiOjE0NDgwMDE0NDMsImF1ZCI6ImtBMzNWSHBIV1ZJSUotZXIyVVVodmVwNTZLcm5iMGhmIn0.FUfhN28mAG6_xpSShae8wvTsIcXaH6eA19d056YooD8LTfdxm3vkyLTpm8buiAov9sJY3ww-F6xcKRlyNn9vAzN68jieOqZycJH4XDBh3jKP-qTuc__6N0jbTY4LmWmuCj0qk2oT6g7ERooL7JLKWFNggf9qQYyuX5JB9kJWIzbvB2bHr5ZopCEg6x0pLp1dFmvbrxDmx_QcV_poqA18RKrdvHJ-HgKbTIlGFRHGqg6Wjh5hUtOMOL1-JeCJHvc7DrqDNgVA1uo_GDPpO5a-lWSSwEVSET76A8kNu0JO-ewIZSjJh3MfoGa6Fi9cTx1Vk6gyXfYQyvcSuTC0OFCWFg"
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
	ride_req := &Ride_request{}
	ride_req.Pid = productid
	ride_req.StartLat = user.Cc.Lat
	ride_req.StartLng = user.Cc.Lng
	err = collection.Find(bson.M{"_id":oid2}).One(&user)
    if err != nil {
    	log.Println("Record not found",err)
    }  
	ride_req.EndLat = user.Cc.Lat
	ride_req.EndLng = user.Cc.Lng
	buf, _ := json.Marshal(ride_req)
	body := bytes.NewBuffer(buf)
	url := fmt.Sprintf(APIUrl, "requests?","access_token=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSIsInJlcXVlc3QiLCJoaXN0b3J5Il0sInN1YiI6ImMxZmNiNzdjLTI5MDEtNDY2Zi05ZTIyLTFlMTExNmFhZWQwNiIsImlzcyI6InViZXItdXMxIiwianRpIjoiOTY1ZTFjNjgtOTY5Ni00ZDQ1LWFlMWQtMTk5MjU5NzQ2ZDJiIiwiZXhwIjoxNDUwNTkzNTM0LCJpYXQiOjE0NDgwMDE1MzMsInVhY3QiOiJ3R1JyMjlLVGRFdUNHY2ExcHJtVFZRQmhjWmlEc3UiLCJuYmYiOjE0NDgwMDE0NDMsImF1ZCI6ImtBMzNWSHBIV1ZJSUotZXIyVVVodmVwNTZLcm5iMGhmIn0.FUfhN28mAG6_xpSShae8wvTsIcXaH6eA19d056YooD8LTfdxm3vkyLTpm8buiAov9sJY3ww-F6xcKRlyNn9vAzN68jieOqZycJH4XDBh3jKP-qTuc__6N0jbTY4LmWmuCj0qk2oT6g7ERooL7JLKWFNggf9qQYyuX5JB9kJWIzbvB2bHr5ZopCEg6x0pLp1dFmvbrxDmx_QcV_poqA18RKrdvHJ-HgKbTIlGFRHGqg6Wjh5hUtOMOL1-JeCJHvc7DrqDNgVA1uo_GDPpO5a-lWSSwEVSET76A8kNu0JO-ewIZSjJh3MfoGa6Fi9cTx1Vk6gyXfYQyvcSuTC0OFCWFg")
	res, err := http.Post(url,"application/json",body)
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	var ride_response Ride_request_response
   	err = json.Unmarshal(data, &ride_response)
	res.Body.Close()
	return ride_response.Eta

}

func find_record(oid bson.ObjectId) Trip{
	sess, err := mgo.Dial("mongodb://Richa:Indore#1@ds041934.mongolab.com:41934/assignment_2_db")
    if err != nil {
       fmt.Printf("Can't connect to mongo, go error %v\n", err)
       os.Exit(1)
    }
    defer sess.Close()
    sess.SetSafe(&mgo.Safe{})
    collection := sess.DB("assignment_2_db").C("routes")
    var get_response Trip
    err = collection.Find(bson.M{"_id":oid}).One(&get_response)
    if err != nil {
    	log.Println("Record not found",err)
    }  
    return get_response

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
	options.AccessToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzY29wZXMiOlsicHJvZmlsZSJdLCJzdWIiOiJjMWZjYjc3Yy0yOTAxLTQ2NmYtOWUyMi0xZTExMTZhYWVkMDYiLCJpc3MiOiJ1YmVyLXVzMSIsImp0aSI6IjYzNzhiMGNjLWZjMTktNDcyMC05ODUxLTJlNmU2YWU1Mjc3OSIsImV4cCI6MTQ1MDUwNzc3NCwiaWF0IjoxNDQ3OTE1NzczLCJ1YWN0IjoiQjNlSzF4ck5IR2lpcG0wbnVJVGo5bGdqazV3eGQxIiwibmJmIjoxNDQ3OTE1NjgzLCJhdWQiOiJrQTMzVkhwSFdWSUlKLWVyMlVVaHZlcDU2S3JuYjBoZiJ9.CWWNX2ElfbqQYJcu_YRsTUiqkSBBlMd6BoOnjroLoYHfnNxFeJI61CU8Bd7qkY2Z_u8WC3lkWJR-3WtJiTDdD-kph-Imw_Nqa3XQalrpsGwCyOvXS57_zWYb8v9E3o3GnRwsDr6TeoEmDkEdikKsK3RlQ6W-U9ywIE3IXXFwaJZyAqS-Q8U_D6NxZb0CQF8212pJSGn34cXEucRtiBJPzuS1VsXLcd6Tgk2AcOXlUU43ukzU_UCK5YBDWvbSXgDISMqBkZHFgxMSubWyTAtdEGtjL-N1Zm9TG-bB66mcPCvAeeg8aam48NX7yGCVasH6z2-U2FCjXSr00rKUcapleQ"
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
	var uber_response Uber_api_response
	for _, price := range pe.Prices {
		if price.ProductId == productid{
		uber_response.Uber_cost = price.LowEstimate
		uber_response.Uber_duration= price.Duration
		uber_response.Uber_distance = price.Distance
		}
	}
	return uber_response
}


func Insert_to_mongodb(final_response Trip){
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


func Update_mongodb(trip Trip) Trip{
		//log.Println ("trip dta: ",trip)
  		sess, err := mgo.Dial("mongodb://Richa:Indore#1@ds041934.mongolab.com:41934/assignment_2_db")
  		if err != nil {
    		fmt.Printf("Can't connect to mongo, go error %v\n", err)
    		os.Exit(1)
  		}
  		defer sess.Close()
  		sess.SetSafe(&mgo.Safe{})
  		collection := sess.DB("assignment_2_db").C("routes")
  		var trip_update Trip
    	err = collection.Remove(bson.M{"_id":trip.Id_put})
    	if err != nil {
    	// handle error
    		fmt.Printf("Delete Error : ", err)
    	}
    	err = collection.Insert(trip)
  		if (err != nil ) {
  			log.Println("error in inserting to database",err)
        
    	}
    	err = collection.Find(bson.M{"_id":trip.Id_put}).One(&trip_update)
    	if err != nil {
    	// handle error
      		fmt.Println("Select record error ",err)
    	
    	}
    	return trip_update



}