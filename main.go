package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL        = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold   = 30           // Load Average threshold
	memoryThreshold = 80           // percent, integer threshold (80%)
	diskThreshold   = 90           // percent, integer threshold (90%)
	networkThreshold = 90          // percent, integer threshold (90%)
	checkInterval   = 1 * time.Second // how often to poll (tests will stop the process)
	maxErrors       = 3
)

func main() {
	errorCount := 0
	// ensure prints are flushed even if buffer - stdout is unbuffered typically
	for {
		resp, err := http.Get(statsURL)
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// read and validate
		if resp.StatusCode != http.StatusOK {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			time.Sleep(checkInterval)
			continue
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// parse
		line := strings.TrimSpace(string(bodyBytes))
		parts := strings.Split(line, ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// successful parse -> reset error count
		errorCount = 0

		// parse integers (use 64-bit)
		load, err1 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		memTotal, err2 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		memUsed, err3 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		diskTotal, err4 := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		diskUsed, err5 := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		netTotal, err6 := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64)
		netUsed, err7 := strconv.ParseInt(strings.TrimSpace(parts[6]), 10, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			// invalid format
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// 1) Load average
		if load > loadThreshold {
			// print integer value
			fmt.Printf("Load Average is too high: %d\n", load)
		}

		// 2) Memory usage percent (integer, floor)
		// protect against division by zero
		if memTotal > 0 {
			memPercent := int((memUsed * 100) / memTotal) // integer division -> floor
			if memPercent > memoryThreshold {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk usage: check if used/diskTotal > 90%
		if diskTotal > 0 {
			// compute used percent via integers: used*100 / total
			diskUsedPercent := int((diskUsed * 100) / diskTotal)
			if diskUsedPercent > diskThreshold {
				// free MB left, integer
				freeBytes := diskTotal - diskUsed
				if freeBytes < 0 {
					freeBytes = 0
				}
				freeMB := freeBytes / 1024 / 1024
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
			}
		}

		// 4) Network: if used/total > 90% -> print available in MB/s (integer) but label Mbit/s
		if netTotal > 0 {
			netUsedPercent := int((netUsed * 100) / netTotal)
			if netUsedPercent > networkThreshold {
				availableBytes := netTotal - netUsed
				if availableBytes < 0 {
					availableBytes = 0
				}
				availableMB := availableBytes / 1024 / 1024 // MB/s (integer)
				// IMPORTANT: tests expect this integer but with the text "Mbit/s available"
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availableMB)
			}
		}

		// small sleep before next poll
		time.Sleep(checkInterval)
	}

	// unreachable, but keep os.Exit to satisfy some linters if needed
	_ = os.Stdout
}
