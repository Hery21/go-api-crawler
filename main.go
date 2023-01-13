package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Sample URL: https://xkcd.com/571/info.0.json
const Url = "https://xkcd.com"
const timeOut = 1

var jobs = make(chan int, 100) // Buffered channel for 100 data
var results = make(chan Result, 100)
var resultCollection []Result

func fetch(n int) (*Result, error) {

	client := &http.Client{
		Timeout: timeOut * time.Minute,
	}

	url := strings.Join([]string{Url, fmt.Sprintf("%d", n), "info.0.json"}, "/")

	req, err := http.NewRequest("GET", url, nil)

	if err != nil {
		return nil, fmt.Errorf("http request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http err: %v", err)
	}

	var data Result

	// error from web service, empty struct to avoid disruption of process
	if resp.StatusCode != http.StatusOK {
		data = Result{}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return nil, fmt.Errorf("json err: %v", err)
		}
	}

	resp.Body.Close()

	return &data, nil
}

// Assign a sequential number (1,2,3 ...) for each of the worker
func allocateJobs(noOfJobs int) {
	for i := 0; i < noOfJobs; i++ {
		jobs <- i + 1
	}

	close(jobs) // stop expectation, memory leak
}

// Loop to fetch according to the number of jobs
func worker(wg *sync.WaitGroup) {
	for job := range jobs {
		result, err := fetch(job)

		if err != nil {
			log.Printf("Error in fetching")
		}
		results <- *result
	}
	wg.Done()
}

// Create a work group of goroutines
func createWorkerPool(noOfWorkers int) {
	var wg sync.WaitGroup

	for i := 0; i <= noOfWorkers; i++ {
		wg.Add(1)
		go worker(&wg)
	}
	wg.Wait()
	close(results)
}

// Put all the separate result into an array of results
func getResults(done chan bool) {
	for result := range results {
		if result.Num != 0 {
			fmt.Println("Rertrieving fata from number", result.Num)
			resultCollection = append(resultCollection, result)
		}
	}
	done <- true
}

// Write the data into a JSON file
func writeToFile(data []byte, fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	var noOfJobs int
	start := time.Now()
	noOfWorkers := 100

	fmt.Println("How many data that you want to get?")
	fmt.Scanln(&noOfJobs)

	go allocateJobs(noOfJobs)

	done := make(chan bool)
	go getResults(done) // mengandalkan worker()

	createWorkerPool(noOfWorkers) // karena bukan goroutine jadi posisinya stlh goroutine
	// jika ditaruh di atas goroutine maka belum ada yang menunggu

	<-done // menunggu done baru lanjut (ada di getresult())

	data, err := json.MarshalIndent(resultCollection, "", "    ")

	if err != nil {
		log.Fatal("json err: ", err)
	}
	err = writeToFile(data, "comics.json")

	fmt.Println("Time elapsed:", time.Since(start))
}
