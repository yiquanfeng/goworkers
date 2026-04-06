package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

type Job struct {
	id  int
	url string
}

type Result struct {
	job     Job
	workid  int
	latency time.Duration
	Success bool
}

func worker(id int, jobs chan Job, results chan Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		var result Result

		fmt.Println("check in")
		spend := rand.Intn(700) + 100
		if spend <= 500 {
			result.Success = true
		} else {
			result.Success = false
		}
		time.Sleep(result.latency)
		result.latency = time.Duration(spend) * time.Microsecond
		result.job = job
		result.workid = id
		results <- result
	}
}

var urls = []string{
	"https://github.com",
	"https://google.com",
	"https://example.com",
	"https://golang.org",
	"https://anthropic.com",
	"https://stackoverflow.com",
	"https://reddit.com",
	"https://twitter.com",
}

func main() {
	jobs := make(chan Job, len(urls))
	results := make(chan Result, len(urls))
	var wg sync.WaitGroup

	wg.Add(3)
	go worker(1, jobs, results, &wg)
	go worker(2, jobs, results, &wg)
	go worker(3, jobs, results, &wg)

	for i := 0; i < len(urls); i++ {
		job := Job{i, urls[i]}
		jobs <- job
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		fmt.Println(r)
	}

}
