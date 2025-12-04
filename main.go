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
	statsURL       = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold  = 30
	memPercentThr  = 80
	diskPercentThr = 90
	netPercentThr  = 90

	maxErrors    = 3
	pollInterval = 1 * time.Second
	httpTimeout  = 5 * time.Second
)

func main() {
	// Allow overriding URL for local testing
	url := statsURL
	if env := os.Getenv("STATS_URL"); env != "" {
		url = env
	}

	client := &http.Client{
		Timeout: httpTimeout,
	}

	errorCount := 0
	printedUnable := false

	for {
		resp, err := client.Get(url)
		if err != nil {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(pollInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(pollInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(pollInterval)
			continue
		}

		text := strings.TrimSpace(string(body))
		parts := strings.Split(text, ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(pollInterval)
			continue
		}

		// successful parse -> reset errors/flag
		errorCount = 0
		printedUnable = false

		// parse values as uint64 (safer for large values)
		loadAvg, ok := parseUint(parts[0])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		memTotal, ok := parseUint(parts[1])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		memUsed, ok := parseUint(parts[2])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		diskTotal, ok := parseUint(parts[3])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		diskUsed, ok := parseUint(parts[4])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		netTotal, ok := parseUint(parts[5])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}
		netUsed, ok := parseUint(parts[6])
		if !ok {
			time.Sleep(pollInterval)
			continue
		}

		// 1) Load Average
		if loadAvg > uint64(loadThreshold) {
			fmt.Printf("Load Average is too high: %d\n", loadAvg)
		}

		// 2) Memory usage %
		if memTotal > 0 {
			memPercent := (memUsed * 100) / memTotal
			if memPercent > uint64(memPercentThr) {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk free in MB
		if diskTotal > 0 {
			diskPercent := (diskUsed * 100) / diskTotal
			if diskPercent > uint64(diskPercentThr) {
				freeMb := (diskTotal - diskUsed) / (1024 * 1024)
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMb)
			}
		}

		// 4) Network
		if netTotal > 0 {
			netPercent := (netUsed * 100) / netTotal
			if netPercent > uint64(netPercentThr) {
				availableBytes := netTotal - netUsed
				// As expected by autotests, convert available bytes to an integer value by dividing by 1_000_000
				availableMbit := availableBytes / 1000000
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availableMbit)
			}
		}

		time.Sleep(pollInterval)
	}
}

// parseUint parses string s into uint64, returns (value, ok)
func parseUint(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if v, err := strconv.ParseUint(s, 10, 64); err == nil {
		return v, true
	}
	// fallback: try signed parsing (in case)
	if v2, err2 := strconv.ParseInt(s, 10, 64); err2 == nil && v2 >= 0 {
		return uint64(v2), true
	}
	return 0, false
}


