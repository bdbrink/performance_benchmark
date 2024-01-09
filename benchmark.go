package main

import (
	"flag"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var (
    server   = flag.String("server", "", "URL of the server to benchmark")
    concurrency = flag.Int("concurrency", 10, "Number of concurrent requests")
    duration = flag.Duration("duration", 10*time.Second, "Duration of the benchmark test")
)

func main() {
    flag.Parse()

    // Error handling for missing server flag
    if *server == "" {
        fmt.Println("Please specify the server URL using the -server flag")
        return
    }

    // Benchmark function
    benchmark()
}

func benchmark() {
    startTime := time.Now()

    var wg sync.WaitGroup
    wg.Add(*concurrency)

    for i := 0; i < *concurrency; i++ {
        go func() {
            defer wg.Done()

            for {
                // Send request to the server
                resp, err := http.Get(*server)

                // Error handling
                if err != nil {
                    fmt.Println("Error:", err)
                    continue
                }

                // Measure response time
                responseTime := time.Since(startTime)
                fmt.Println(responseTime)

                // Collect response time statistics
                // ... (implement later)

                resp.Body.Close()

                // Check if benchmark duration has elapsed
                if time.Since(startTime) > *duration {
                    break
                }
            }
        }()
    }

    wg.Wait()

}
