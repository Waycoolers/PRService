package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"
)

type PRCreateReq struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type PRReassignReq struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

func printStats(times []time.Duration, success, fail int) {
	sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
	total := time.Duration(0)
	for _, t := range times {
		total += t
	}
	avg := total.Seconds() * 1000 / float64(len(times))
	p95 := times[int(math.Ceil(0.95*float64(len(times))))-1].Seconds() * 1000
	p99 := times[int(math.Ceil(0.99*float64(len(times))))-1].Seconds() * 1000

	fmt.Printf("Requests: %d success, %d fail, avg=%.2fms, p95=%.2fms, p99=%.2fms\n",
		success, fail, avg, p95, p99)
}

func main() {
	baseURL := "http://localhost:8080"
	numRequests := 1000
	concurrency := 50

	fmt.Println("Starting CreatePR load test...")
	runLoadTest(numRequests, concurrency, func(i int) (string, error) {
		pr := PRCreateReq{
			PullRequestID:   fmt.Sprintf("pr-load-%d", i),
			PullRequestName: fmt.Sprintf("Load PR %d", i),
			AuthorID:        fmt.Sprintf("u1_%d", i%5+1),
		}
		data, _ := json.Marshal(pr)
		start := time.Now()
		resp, err := http.Post(baseURL+"/pullRequest/create", "application/json", bytes.NewBuffer(data))
		if resp != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Printf("error closing response body: %v", err)
				}
			}(resp.Body)
		}
		return fmt.Sprintf("%v", time.Since(start)), err
	})

	fmt.Println("Starting ReassignReviewer load test...")
	runLoadTest(numRequests, concurrency, func(i int) (string, error) {
		req := PRReassignReq{
			PullRequestID: fmt.Sprintf("pr-load-%d", i%numRequests),
			OldUserID:     fmt.Sprintf("u1_%d", i%5+1),
		}
		data, _ := json.Marshal(req)
		start := time.Now()
		resp, err := http.Post(baseURL+"/pullRequest/reassign", "application/json", bytes.NewBuffer(data))
		if resp != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Printf("error closing body: %v", err)
				}
			}(resp.Body)
		}
		return fmt.Sprintf("%v", time.Since(start)), err
	})

	fmt.Println("Starting Stats load test...")
	runLoadTest(numRequests, concurrency, func(i int) (string, error) {
		start := time.Now()
		resp, err := http.Get(baseURL + "/stats")
		if resp != nil {
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Printf("error closing resp.Body: %v", err)
				}
			}(resp.Body)
		}
		return fmt.Sprintf("%v", time.Since(start)), err
	})
}

func runLoadTest(numRequests, concurrency int, f func(int) (string, error)) {
	var wg sync.WaitGroup
	times := make([]time.Duration, numRequests)
	success := 0
	fail := 0
	mu := sync.Mutex{}

	jobs := make(chan int, numRequests)
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				tStr, err := f(i)
				t, _ := time.ParseDuration(tStr)
				mu.Lock()
				times[i] = t
				if err != nil {
					fail++
				} else {
					success++
				}
				mu.Unlock()
			}
		}()
	}

	for i := 0; i < numRequests; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	printStats(times, success, fail)
}
