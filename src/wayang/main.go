package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"net/http"
	"strings"
	"wayang/wayang"
)

type Config struct {
	Database        string `json:"db"`
	DatabaseAddress string `json:"db_addr"`
	StaticConf      string `json:"static_conf"`
}

var (
	db wayang.DataStore
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
	config := Config{}

	err = json.Unmarshal(confData, &config)
	if err != nil {
		fmt.Println("Failed to parse configuration file:", *configPath)
		return
	}

	r := mux.NewRouter()
	r.StrictSlash(true)

	if config.Database == "mongodb" {

		db, err = wayang.NewMongoStore(config.DatabaseAddress)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.HandleFunc("/", indexPage).Methods("GET")
		r.HandleFunc("/", newMock).Methods("POST")
		r.HandleFunc("/{id:[0-9a-z]+}", optionsHandlerRoot).Methods("OPTIONS")
		r.HandleFunc("/{id:[0-9a-z]+}/{endpoint:[a-zA-Z0-9/]+}", optionsHandler).Methods("OPTIONS")
		r.HandleFunc("/{id:[0-9a-z]+}", mockRespondRoot).Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		r.HandleFunc("/{id:[0-9a-z]+}/{endpoint:[a-zA-Z0-9/]+}", mockRespond).Methods("GET", "POST", "PUT", "PATCH", "DELETE")

	} else if config.Database == "static" {

		mock := wayang.Mock{}
		mockData, err := ioutil.ReadFile(config.StaticConf)
		if err != nil {
			fmt.Println("Failed to read static configuration file:", config.StaticConf)
			return
		}
		err = json.Unmarshal(mockData, &mock)
		if err != nil {
			fmt.Println("Failed to parse static configuration file:", config.StaticConf)
			return
		}

		db, err = wayang.NewStaticStore(mock)
		if err != nil {
			fmt.Println(err)
			return
		}

		r.HandleFunc("/", optionsHandlerRoot).Methods("OPTIONS")
		r.HandleFunc("/{endpoint:[a-zA-Z0-9/]+}", optionsHandler).Methods("OPTIONS")
		r.HandleFunc("/", mockRespondRoot).Methods("GET", "POST", "PUT", "PATCH", "DELETE")
		r.HandleFunc("/{endpoint:[a-zA-Z0-9/]+}", mockRespond).Methods("GET", "POST", "PUT", "PATCH", "DELETE")

	}
	defer db.Close()

	http.Handle("/", r)
	http.ListenAndServe(*listenAddr, nil)

}

func accessControlAllow(rw http.ResponseWriter, req *http.Request) {
	origin := req.Header.Get("Origin")
	if origin == "" {
		rw.Header().Set("Access-Control-Allow-Origin", "*")
	} else {
		rw.Header().Set("Access-Control-Allow-Origin", origin)
	}
}

func indexPage(rw http.ResponseWriter, req *http.Request) {
	accessControlAllow(rw, req)
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

func newMock(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	accessControlAllow(rw, req)
	rw.Header().Set("Content-Type", "application/json")
	resp := NewMockResponse{}

	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.URL = ""
		resp.Status = http.StatusText(http.StatusInternalServerError)
		resp.Detail = err.Error()
		rv, _ := json.Marshal(resp)
		http.Error(rw, string(rv), http.StatusInternalServerError)
		return
	}
	mock := wayang.Mock{}
	err = json.Unmarshal(body, &mock)
	if err != nil {
		resp.URL = ""
		resp.Status = http.StatusText(http.StatusBadRequest)
		resp.Detail = err.Error()
		rv, _ := json.Marshal(resp)
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
				resp.URL = ""
				resp.Status = http.StatusText(http.StatusBadRequest)
				resp.Detail = "Invalid key " + kk + " in JSON."
				rv, _ := json.Marshal(resp)
				http.Error(rw, string(rv), http.StatusBadRequest)
				return
			}
			if kk != KK {
				mock[k][KK] = vv
				delete(mock[k], kk)
			}
		}
	}

	fmt.Println(mock)

	id, err := db.NewMock(mock)
	if err != nil {
		resp.URL = ""
		resp.Status = http.StatusText(http.StatusInternalServerError)
		resp.Detail = err.Error()
		rv, _ := json.Marshal(resp)
		http.Error(rw, string(rv), http.StatusInternalServerError)
		return
	} else {
		resp.Status = "200 OK"
	}
	resp.URL = "http://" + req.Host + "/" + id

	rv, _ := json.Marshal(resp)

	rw.Write(rv)
}

func optionsHandlerRoot(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/"

	accessControlAllow(rw, req)
	processOptionsResponse(rw, req, id, endpoint)
}

func optionsHandler(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/" + vars["endpoint"]

	accessControlAllow(rw, req)
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

	accessControlAllow(rw, req)
	processEndpointResponse(rw, req, id, endpoint)
}

func mockRespond(rw http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	id := vars["id"]
	endpoint := "/" + vars["endpoint"]

	accessControlAllow(rw, req)
	processEndpointResponse(rw, req, id, endpoint)
}

func processEndpointResponse(rw http.ResponseWriter, req *http.Request, id string, endpoint string) {

	defer req.Body.Close()

	accessControlAllow(rw, req)
	ep, err := db.GetEndpoint(id, endpoint)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "application/json")
	mockResponse := ep[req.Method]

	if len(mockResponse) == 0 {
		http.Error(rw, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	rv, _ := json.Marshal(mockResponse)
	rw.Write([]byte(rv))

}
