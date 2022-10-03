package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gammazero/workerpool"
)

// local
var baseLocalUrl = "http://localhost:8080"

// express
var expressEndpoint = "/api/v1/express-animals?limit=%v"
var expressBaseURL = "https://express-animals-production.up.railway.app"

// go
var goEndpoint = "/v1/go-animals?limit=%v"
var goBaseURL = "https://animals-production.up.railway.app"

func main() {
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)

	url := expressBaseURL + expressEndpoint
	//url := goBaseURL + goEndpoint

	wp := workerpool.New(200)
	start := time.Now()
	loops := 20000
	httpErrs := make([]error, 0, 100)
	badResponse := make([]map[string]interface{}, 0, 100)
	nilRepsonse := make([]map[string]interface{}, 0, 100)

	buckets := Buckets{
		Mu:    &sync.Mutex{},
		Bucks: map[string]int{},
	}

	for i := 0; i < loops; i++ {
		wp.Submit(func() {
			limit := r1.Intn(100)
			limit++
			reqStart := time.Now()
			resp, err := makeRequest(limit, url)
			if err != nil {
				httpErrs = append(httpErrs, err)
				log.Fatalln("error getting data", err)
			}

			elapsed := time.Since(reqStart)
			if resp != nil {
				animals, ok := resp["animals"].([]interface{})
				if ok {
					log.Println("len of animals:", len(animals), elapsed)
				}
			} else {
				nilRepsonse = append(nilRepsonse, resp)
			}

			elapsedSplit := strings.Split(elapsed.String(), ".")
			buckets.BucketTime(elapsedSplit[0])
		})
	}
	wp.StopWait()

	elapsed := time.Since(start)
	rps := loops / int(elapsed.Seconds())
	log.Println("rps: ", rps)
	log.Println("error count:", len(httpErrs))
	log.Println("bad response count:", len(badResponse))
	log.Println("nil response count:", len(nilRepsonse))
	for key, value := range buckets.Bucks {
		log.Println("key, value", key, value)
	}
}

type Buckets struct {
	Mu    *sync.Mutex
	Bucks map[string]int
}

func (b *Buckets) BucketTime(time string) {
	timeInt, _ := strconv.Atoi(time)

	b.Mu.Lock()
	defer b.Mu.Unlock()

	switch {
	case timeInt < 1 && timeInt < 50:
		b.Bucks["0_to_50"]++
	case timeInt < 51 && timeInt < 200:
		b.Bucks["51_to_200"]++
	case timeInt < 201 && timeInt < 500:
		b.Bucks["201_to_500"]++
	case timeInt < 501 && timeInt < 1000:
		b.Bucks["500_to_1000"]++
	case timeInt < 1001 && timeInt < 1500:
		b.Bucks["1000_to_1500"]++
	default:
		b.Bucks["uncategorized"]++
	}
}

func makeRequest(limit int, url string) (map[string]interface{}, error) {

	resp, err := http.DefaultClient.Get(fmt.Sprintf(url, limit))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > 299 {
		return nil, fmt.Errorf("bad status code %v", resp.StatusCode)
	}

	response := map[string]interface{}{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		log.Println("faield to decode body")
		return nil, err
	}

	return response, nil
}
