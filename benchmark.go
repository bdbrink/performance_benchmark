package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
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

    go trackResourceUsage()
    benchmark()
}

func benchmark() {
    startTime := time.Now()
    var responseTimes []time.Duration
    var successfulRequests int
    var failedRequests int

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
                    failedRequests++
                    fmt.Println("Error:", err)
                    continue
                }

                // Measure response time
                responseTime := time.Since(startTime)
                responseTimes = append(responseTimes, responseTime)

                resp.Body.Close()
                successfulRequests++

                // Check if benchmark duration has elapsed
                if time.Since(startTime) > *duration {
                    break
                }
            }
        }()
    }

    wg.Wait()

    // Calculate and print response time statistics
    sort.Slice(responseTimes, func(i, j int) bool {
        return responseTimes[i] < responseTimes[j]
    })

    mean := time.Duration(0)
    for _, rt := range responseTimes {
        mean += rt
    }
    mean /= time.Duration(len(responseTimes))

    median := responseTimes[len(responseTimes)/2]
    p95 := responseTimes[int(0.95*float64(len(responseTimes)))]

    fmt.Printf("\nResponse Time Statistics:\n")
    fmt.Printf("Mean: %v\n", mean)
    fmt.Printf("Median: %v\n", median)
    fmt.Printf("95th Percentile: %v\n", p95)

    // Calculate and print throughput
    throughput := float64(successfulRequests) / duration.Seconds()
    fmt.Printf("\nThroughput: %.2f requests/second\n", throughput)

    // Print error statistics
    fmt.Printf("\nError Statistics:\n")
    fmt.Printf("Failed Requests: %d\n", failedRequests)
}


func trackResourceUsage() {
    var beginningMem runtime.MemStats
    runtime.ReadMemStats(&beginningMem)
    startTime := time.Now()

    go func() {
        for {
            // Collect CPU usage
            cpuUsage, err := cpu.Percent(time.Second, false)
            if err != nil {
                fmt.Println("Error getting CPU usage:", err)
                continue
            }

            // Collect memory usage
            var currentMem runtime.MemStats
            runtime.ReadMemStats(&currentMem)

            // Print or save resource usage metrics
            fmt.Printf("CPU Usage: %.2f%%\n", cpuUsage[0])
            fmt.Printf("Memory Usage: %d MB\n", currentMem.Alloc/1024/1024)

            // Check if benchmark duration has elapsed
            if time.Since(startTime) > *duration {
                break
            }

            time.Sleep(time.Second) // Adjust interval as needed
        }
    }()
}

func monitorNetwork() {
    var wg sync.WaitGroup
    wg.Add(1)

    go func() {
        defer wg.Done()

        startTime := time.Now()
        var bytesSent int64
        var bytesReceived int64
        var connectionsOpened int64
        var connectionErrors int64

        for {
            // Use net package functions to get network metrics
            // (Implementation depends on your specific needs)

            // Example using net/http/pprof:
            pprofStats := new(pprof.Profile).Count()
            bytesSent += pprofStats.BytesSent
            bytesReceived += pprofStats.BytesReceived
            connectionsOpened += pprofStats.ConnsCreated

            // Print or save network metrics
            fmt.Printf("\nNetwork Metrics:\n")
            fmt.Printf("Bytes Sent: %d\n", bytesSent)
            fmt.Printf("Bytes Received: %d\n", bytesReceived)
            fmt.Printf("Connections Opened: %d\n", connectionsOpened)
            fmt.Printf("Connection Errors: %d\n", connectionErrors)

            // Check if benchmark duration has elapsed
            if time.Since(startTime) > *duration {
                break
            }

            time.Sleep(time.Second) // Adjust interval as needed
        }
    }()

    wg.Wait()
}

