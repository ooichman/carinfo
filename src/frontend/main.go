package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"net/http"
	"log"
	"encoding/json"
	"bytes"
)


type CarInfoRequest struct {

	Module string `json:module`
	Manufacture string `json:manufacture`
	
}

type CarInfoAnswer struct {

	Name string `json:name`
	Year int `json:year`
	Condition string `json:condition`
	Reason string `json:reason`
	Module string `json:module`
	Manufacture string `json:manufacture`

}

func getEnv(key, fallback string) string {
	value , exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}

	return value
}

func QueryRes(w http.ResponseWriter, r *http.Request) {

	var requestdata CarInfoRequest

	if r.URL.Path != "/query" {
		http.Error(w, "the Provided URL is invalid\n", http.StatusBadRequest)
		fmt.Fprintf(os.Stderr, "The provided URL is invalid\n")

	}

	if r.Method != "POST" {
		http.Error(w,"The HTTP Method Must be POST\n", http.StatusBadRequest)
		fmt.Fprintf(os.Stderr,"The HTTP Method Must be POST\n")
	}

	var Body []byte

	if r.Body != nil {
		if data , err := ioutil.ReadAll(r.Body); err == nil {
			Body = data
		} else {

			fmt.Fprintf(os.Stderr, "Unable to Copy the Body\n")
			http.Error(w,"Unable to copy the Body\n", http.StatusBadRequest)
		}
	}

	if err := json.Unmarshal(Body, &requestdata ); err != nil {

		fmt.Fprintf(os.Stderr, "Unable to Unmarshal the body\n")
		http.Error(w, "Unable to Unmarshal the body\n", http.StatusBadRequest)
		return
	} else {
		fmt.Fprintf(os.Stdout,"%+v\n", requestdata)
	}

    sendBody, _ := json.Marshal(requestdata)

    serviceAddr := getEnv("DBAPI_URL", "http://dbapi:8080/query")
    response , re_err := http.Post(serviceAddr , "application/json", bytes.NewBuffer(sendBody))

    if re_err != nil {
    	fmt.Fprintf(os.Stderr, "Unable to Retreive data from DB API\n")
    	http.Error(w, "Unable to Retreive data from DB API\n", http.StatusBadRequest)
    }

    defer response.Body.Close()

    if response.StatusCode == http.StatusOK {
    	resbody , err := ioutil.ReadAll(response.Body)

    	if err != nil {
    		fmt.Fprintf(os.Stderr, "Unable to Copy response Body\n")
    		http.Error(w, "Unable to Copy response Body\n", http.StatusBadRequest)
    	}

    	if _ , werr := w.Write(resbody); werr != nil {
    		fmt.Fprintf(os.Stderr, "Can't write a response\n")
    		http.Error(w, "Can't write a response\n", http.StatusBadRequest)
    	}
    } else {
    	fmt.Fprintf(os.Stderr, "received a bad response from DB API\n")
    	http.Error(w, "received a bad response from DB API\n", http.StatusBadRequest)
    }

}

func StaticRes(w http.ResponseWriter, r *http.Request) {

	ContentType := r.Header.Get("Content-type")
	var answerdata CarInfoAnswer
	//var requestdata CarInfoRequest

	if r.Method != "GET" {
		fmt.Fprintf(w,"Unale able to hendle none GET requests\n")
		fmt.Fprintf(os.Stderr,"Unale able to hendle none GET requests\n")
	
		return
	}

	if r.URL.Path != "/static" {
		fmt.Fprintf(w,"URL provided is invalid\n")
		fmt.Fprintf(os.Stderr,"URL Provided is invalid")
		return
	}



	if ContentType == "application/json" {

		fmt.Fprintf(os.Stdout, "Content Type JSON received\n")

		answerdata.Name = "Example"
		answerdata.Year = 1999
		answerdata.Condition = "New"
		answerdata.Reason = "Static test"
		answerdata.Module = "Spider"
		answerdata.Manufacture = "Alfa Romeo"

		sendAnswer , errAnswer := json.Marshal(answerdata)

		if errAnswer != nil {
			fmt.Fprintf(os.Stderr,"Unable to Marshel the Request\n")
			fmt.Fprintf(w,"Unable to Marshel the Request\n")
			return
		}

		if _ , werr := w.Write(sendAnswer); werr != nil {
			fmt.Fprintf(os.Stderr,"Unable to Write a response\n")
			http.Error(w, "Unable to Write A response", http.StatusBadRequest)
		}
	} else {
		fmt.Fprintf(w,"Content Type Not supported\n")
		http.Error(w, "Content Type Not supported", http.StatusBadRequest)
	}

}

func main() {

	portnum := getEnv("PORT", "8080")
	http.HandleFunc("/static", StaticRes)
	http.HandleFunc("/query", QueryRes)
	log.Printf("Staring HTTP Service on port %v", portnum)
	log.Fatal(http.ListenAndServe(":"+portnum, nil))
	}