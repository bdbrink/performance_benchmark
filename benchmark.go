package main

import (
	"bytes"
	"flag"
	"fmt"
	"gonum/plot/vg"
	"net/http"
	"net/http/pprof"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/shirou/gopsutil/cpu"
)

var (
    server   = flag.String("server", "", "URL of the server to benchmark")
    concurrency = flag.Int("concurrency", 10, "Number of concurrent requests")
    duration = flag.Duration("duration", 10*time.Second, "Duration of the benchmark test")
    method       = flag.String("method", "GET", "HTTP method to use")
    headers      = flag.String("headers", "", "Headers to include in the request (comma-separated key=value pairs)")
    payload      = flag.String("payload", "", "Payload to send with the request")
)

func main() {
    flag.Parse()

    // Error handling for missing server flag
    if *server == "" {
        fmt.Println("Please specify the server URL using the -server flag")
        return
    }

    req, err := createRequest()
    if err != nil {
        fmt.Println("Error creating request:", err)
        return
    }
    resp, err := http.DefaultClient.Do(req)
    fmt.Println(resp)

    go trackResourceUsage()
    go benchmark()
    go monitorNetwork()
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
    p99 := responseTimes[int(0.99*float64(len(responseTimes)))]

    fmt.Printf("\nResponse Time Statistics:\n")
    fmt.Printf("Mean: %v\n", mean)
    fmt.Printf("Median: %v\n", median)
    fmt.Printf("99th Percentile: %v\n", p99)

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

func createRequest() (*http.Request, error) {
    // Parse headers into a map
    headersMap := make(map[string]string)
    if *headers != "" {
        for _, pair := range strings.Split(*headers, ",") {
            kv := strings.Split(pair, "=")
            if len(kv) == 2 {
                headersMap[kv[0]] = kv[1]
            }
        }
    }

    // Create the request with customization
    req, err := http.NewRequest(*method, *server, bytes.NewBufferString(*payload))
    if err != nil {
        return nil, err
    }
    for key, value := range headersMap {
        req.Header.Set(key, value)
    }
    return req, nil
}

func plotResponseTimes(responseTimes []time.Duration, filename string) {
    p, err := plot.New()
    if err != nil {
        fmt.Println("Error creating plot:", err)
        return
    }

    p.Title.Text = "Response Time Distribution"
    p.X.Label.Text = "Response Time (ms)"
    p.Y.Label.Text = "Count"

    // Convert durations to milliseconds
    var msValues []float64
    for _, rt := range responseTimes {
        msValues = append(msValues, float64(rt.Milliseconds()))
    }

    // Create and customize histogram
    hist, err := plotter.NewHist(msValues, 20) // 20 bins
    if err != nil {
        fmt.Println("Error creating histogram:", err)
        return
    }
    hist.Color = plot.Gray{0.4}
    hist.FillStyle = plotter.RectangleStyle{
        Pattern:    plotter.Gray{},
        StrokeColor: plot.Gray{0},
        StrokeWidth: vg.Points(0.5),
    }

    // Add histogram to the plot
    p.Add(hist)

    // Save the plot as a PNG image
    if err := p.Save(filename, svg.Inches(8), svg.Inches(4)); err != nil {
        fmt.Println("Error saving plot:", err)
        return
    }

    fmt.Printf("Saved response time distribution to %s\n", filename)
}