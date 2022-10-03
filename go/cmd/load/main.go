package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/gammazero/workerpool"
)

var endpoint = "/v1/go-animals?limit=%v"
var baseLocalUrl = "http://localhost:8080"
var baseServerUrl = "https://animals-production.up.railway.app"

func main() {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	url := baseServerUrl

	wp := workerpool.New(1)
	start := time.Now()
	loops := 5000
	for i := 0; i < loops; i++ {
		wp.Submit(func() {
			limit := r1.Intn(1000)
			reqStart := time.Now()
			makeRequest(limit, url+endpoint)
			log.Println("elapsed: ", time.Since(reqStart))
		})
	}
	log.Println("bloop")
	wp.StopWait()

	elapsed := time.Since(start)
	rps := loops / int(elapsed.Seconds())
	log.Println("rps: ", rps)
}

func makeRequest(limit int, url string) map[string]interface{} {

	resp, err := http.DefaultClient.Get(fmt.Sprintf(url, limit))
	if err != nil {
		log.Println("err:", err)
		return nil
	}

	if resp.StatusCode > 299 {
		log.Println("bad status code", resp.StatusCode)
		return nil
	}

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println("faield to decode body")
		return nil
	}

	return response
}
