package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"wayang/wayang"
)

type Config struct {
	HTTPPrefix      string `json:"http_prefix"`
	Database        string `json:"db"`
	DatabaseAddress string `json:"db_addr"`
}

var (
	db     wayang.DataStore
	config Config
)

func main() {

	configPath := flag.String("conf", "./config.json", "Path to configuration file")
	listenAddr := flag.String("laddr", ":8000", "Address to listen on")
	flag.Parse()

	confData, err := ioutil.ReadFile(*configPath)
	if err != nil {
		fmt.Println("Failed to read configuration file:", *configPath)
		return
	}
	config.HTTPPrefix = ""

	err = json.Unmarshal(confData, &config)
	if err != nil {
		fmt.Println("Failed to parse configuration file:", *configPath)
		return
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	log.Println("Using", config.Database, "@", config.DatabaseAddress)

	if config.Database == "mongodb" {

		db, err = wayang.NewMongoStore(config.DatabaseAddress)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.HandleFunc(config.HTTPPrefix+"/", rootGet).Methods("GET")
		r.HandleFunc(config.HTTPPrefix+"/", rootPost).Methods("POST")
		r.HandleFunc(config.HTTPPrefix+"/", rootOptions).Methods("OPTIONS")
		r.HandleFunc(config.HTTPPrefix+"/{id:[0-9a-z]+}", optionsHandlerRoot).Methods("OPTIONS")
		r.HandleFunc(config.HTTPPrefix+"/{id:[0-9a-z]+}/{endpoint:[a-zA-Z0-9/_-]+}", optionsHandler).Methods("OPTIONS")
		r.HandleFunc(config.HTTPPrefix+"/{id:[0-9a-z]+}", mockRespondRoot).Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		r.HandleFunc(config.HTTPPrefix+"/{id:[0-9a-z]+}/{endpoint:[a-zA-Z0-9/_-]+}", mockRespond).Methods("GET", "POST", "PUT", "PATCH", "DELETE")

	} else if config.Database == "static" {

		mock := wayang.Mock{}
		mockData, err := ioutil.ReadFile(config.DatabaseAddress)
		if err != nil {
			fmt.Println("Failed to read static configuration file:", config.DatabaseAddress)
			return
		}
		err = json.Unmarshal(mockData, &mock)
		if err != nil {
			fmt.Println("Failed to parse static configuration file:", config.DatabaseAddress)
			return
		}

		db, err = wayang.NewStaticStore(mock)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.HandleFunc(config.HTTPPrefix+"/__config__", staticConfigManagement).Methods("GET", "PUT", "PATCH", "DELETE")
		r.HandleFunc(config.HTTPPrefix+"/", optionsHandlerRoot).Methods("OPTIONS")
		r.HandleFunc(config.HTTPPrefix+"/{endpoint:[a-zA-Z0-9/]+}", optionsHandler).Methods("OPTIONS")
		r.HandleFunc(config.HTTPPrefix+"/", mockRespondRoot).Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		r.HandleFunc(config.HTTPPrefix+"/{endpoint:[a-zA-Z0-9/]+}", mockRespond).Methods("GET", "POST", "PUT", "PATCH", "DELETE")

	}
	defer db.Close()

	http.Handle("/", r)
	http.ListenAndServe(*listenAddr, nil)

}

func accessControlAllow(rw http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin == "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
		log.Println(req.Method, req.RequestURI)
	} else {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
		log.Println(req.Method, req.RequestURI, "[Origin:", req.Header.Get("Origin")+"]")
	}
}

func rootOptions(rw http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	accessControlAllow(rw, req)

	csv := "OPTIONS,GET,POST"
	rw.Header().Set("Allow", csv)
	rw.Header().Set("Access-Control-Allow-Methods", csv)

}

func rootGet(rw http.ResponseWriter, req *http.Request) {
	accessControlAllow(rw, req)
	rw.Header().Set("Content-Type", "text/plain")
	helpMessage := `Hi!
To create an endpoint, do a POST request to the current URL.
The request should contain JSON of the following format:
{
	"/": {
		"GET": {
			"some_key": "some_val"
		},
		"POST": {
			"some_key": {
				"some_other_key": "some_val"
			}
		}
	},
	"/other_endpoint": {
		"DELETE": {
			"key": "value"
		}
	}
}`
	rw.Write([]byte(helpMessage))
}

type NewMockResponse struct {
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
	URL    string `json:"url,omitempty"`
}

func rootPost(rw http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()
	accessControlAllow(rw, req)
	rw.Header().Set("Content-Type", "application/json")

	resp := NewMockResponse{}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.URL = ""
		resp.Status = http.StatusText(http.StatusInternalServerError)
		resp.Detail = err.Error()
		rv, _ := json.MarshalIndent(resp, "", "	")
		http.Error(rw, string(rv), http.StatusInternalServerError)
		return
	}
	mock := unmarshalAndCleanup(rw, body)

	id, err := db.NewMock(mock)
	if err != nil {
		resp.URL = ""
		resp.Status = http.StatusText(http.StatusInternalServerError)
		resp.Detail = err.Error()
		rv, _ := json.MarshalIndent(resp, "", "	")
		http.Error(rw, string(rv), http.StatusInternalServerError)
		return
	} else {
		resp.Status = "200 OK"
	}
	resp.URL = "http://" + req.Host + "/" + id

	rv, _ := json.MarshalIndent(resp, "", "	")

	rw.Write(rv)
}

func optionsHandlerRoot(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/"

	processOptionsResponse(rw, req, id, endpoint)
}

func optionsHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/" + vars["endpoint"]

	processOptionsResponse(rw, req, id, endpoint)
}

func processOptionsResponse(rw http.ResponseWriter, req *http.Request, id string, endpoint string) {

	defer req.Body.Close()
	accessControlAllow(rw, req)

	ep, err := db.GetEndpoint(id, endpoint)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	var allowedMethods []string
	allowedMethods = append(allowedMethods, "OPTIONS")
	for k, _ := range ep {
		allowedMethods = append(allowedMethods, k)
	}

	csv := strings.Join(allowedMethods, ",")
	rw.Header().Set("Allow", csv)
	rw.Header().Set("Access-Control-Allow-Methods", csv)

}

func mockRespondRoot(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/"

	processEndpointResponse(rw, req, id, endpoint)
}

func mockRespond(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/" + vars["endpoint"]

	processEndpointResponse(rw, req, id, endpoint)
}

func processEndpointResponse(rw http.ResponseWriter, req *http.Request, id string, endpoint string) {

	defer req.Body.Close()
	accessControlAllow(rw, req)
	rw.Header().Set("Content-Type", "application/json")

	ep, err := db.GetEndpoint(id, endpoint)
	if err != nil {
		http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	mockResponse, ok := ep[req.Method]
	if !ok {
		http.Error(rw, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	rv, _ := json.MarshalIndent(mockResponse, "", "	")
	rw.Write([]byte(rv))

}

func staticConfigManagement(rw http.ResponseWriter, req *http.Request) {

	defer req.Body.Close()

	accessControlAllow(rw, req)
	rw.Header().Set("Content-Type", "application/json")

	static := db.(*wayang.StaticStore)
	resp := NewMockResponse{}

	if req.Method == "GET" {

		rv, _ := json.MarshalIndent(static.StaticData, "", "	")
		rw.Write([]byte(rv))

	} else if req.Method == "PUT" {

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.Status = strconv.Itoa(http.StatusInternalServerError) + " " + http.StatusText(http.StatusInternalServerError)
			resp.Detail = err.Error()
			rv, _ := json.MarshalIndent(resp, "", "	")
			http.Error(rw, string(rv), http.StatusInternalServerError)
			return
		}

		mock := unmarshalAndCleanup(rw, body)

		static.StaticData = mock

		go syncToFile()

	} else if req.Method == "PATCH" {

		body, err := ioutil.ReadAll(req.Body)
		if err != nil {
			resp.Status = strconv.Itoa(http.StatusInternalServerError) + " " + http.StatusText(http.StatusInternalServerError)
			resp.Detail = err.Error()
			rv, _ := json.MarshalIndent(resp, "", "	")
			http.Error(rw, string(rv), http.StatusInternalServerError)
			return
		}

		partMock := unmarshalAndCleanup(rw, body)

		for k, v := range partMock {
			static.StaticData[k] = v
		}

		go syncToFile()

	} else if req.Method == "DELETE" {

		static.StaticData = wayang.Mock{}

		go syncToFile()

	}

}

func unmarshalAndCleanup(rw http.ResponseWriter, data []byte) (mock wayang.Mock) {

	resp := NewMockResponse{}

	err := json.Unmarshal(data, &mock)
	if err != nil {
		resp.Status = strconv.Itoa(http.StatusBadRequest) + " " + http.StatusText(http.StatusBadRequest)
		resp.Detail = err.Error()
		rv, _ := json.MarshalIndent(resp, "", "	")
		http.Error(rw, string(rv), http.StatusBadRequest)
		return
	}

	// Ensure that all routes begin with a /
	for k, v := range mock {
		if k[:1] != "/" {
			mock["/"+k] = v
			delete(mock, k)
		}
	}

	// Ensure that all requests are in the right format
	for k, v := range mock {
		for kk, vv := range v {
			KK := strings.ToUpper(kk)
			if KK != "GET" &&
				KK != "POST" &&
				KK != "PUT" &&
				KK != "PATCH" &&
				KK != "DELETE" {
				resp.Status = strconv.Itoa(http.StatusBadRequest) + " " + http.StatusText(http.StatusBadRequest)
				resp.Detail = "Invalid HTTP Method"
				rv, _ := json.MarshalIndent(resp, "", "	")
				http.Error(rw, string(rv), http.StatusBadRequest)
				return
			}
			if kk != KK {
				mock[k][KK] = vv
				delete(mock[k], kk)
			}
		}
	}

	return

}

func syncToFile() {
	static := db.(*wayang.StaticStore)
	data, _ := json.MarshalIndent(static.StaticData, "", "	")
	ioutil.WriteFile(config.DatabaseAddress, data, 0644)
}
