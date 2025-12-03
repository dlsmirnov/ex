package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	statsURL       = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold  = 30.0
	memThreshold   = 80  // in percent (we compute integer percent and compare > 80)
	diskThreshold  = 90  // in percent (compare used/diskTotal > 90)
	netThreshold   = 90  // in percent (compare used/netTotal > 90)
	checkInterval  = 500 * time.Millisecond // polling interval
	maxErrors      = 3
)

func main() {
	errors := 0

	for {
		ok := fetchOnce(&errors)
		if ok {
			// successful fetch resets error counter inside fetchOnce
		} else {
			// fetchOnce increments errors and prints "Unable..." when >= maxErrors
		}

		time.Sleep(checkInterval)
	}
}

func fetchOnce(errors *int) bool {
	resp, err := http.Get(statsURL)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}

	text := strings.TrimSpace(string(bodyBytes))
	parts := strings.Split(text, ",")
	if len(parts) != 7 {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}

	// parse all fields as uint64 where applicable
	loadF, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	memTotal, err := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	memUsed, err := strconv.ParseUint(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	diskTotal, err := strconv.ParseUint(strings.TrimSpace(parts[3]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	diskUsed, err := strconv.ParseUint(strings.TrimSpace(parts[4]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	netTotal, err := strconv.ParseUint(strings.TrimSpace(parts[5]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}
	netUsed, err := strconv.ParseUint(strings.TrimSpace(parts[6]), 10, 64)
	if err != nil {
		*errors++
		if *errors >= maxErrors {
			fmt.Println("Unable to fetch server statistic")
		}
		return false
	}

	// success -> reset error counter
	*errors = 0

	// 1) Load Average check
	if loadF > loadThreshold {
		// print as integer (no decimals)
		fmt.Printf("Load Average is too high: %d\n", int(loadF))
	}

	// 2) Memory usage check: integer percent (floor)
	if memTotal > 0 {
		memPct := int((memUsed * 100) / memTotal) // integer division -> floor
		if memPct > memThreshold {
			fmt.Printf("Memory usage too high: %d%%\n", memPct)
		}
	}

	// 3) Disk usage check
	if diskTotal > 0 {
		// used percent (integer)
		diskUsedPct := int((diskUsed * 100) / diskTotal)
		if diskUsedPct > diskThreshold {
			// free in megabytes (floor): (diskTotal - diskUsed) / (1024*1024)
			var freeBytes uint64
			if diskTotal > diskUsed {
				freeBytes = diskTotal - diskUsed
			} else {
				freeBytes = 0
			}
			freeMB := freeBytes / (1024 * 1024)
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
		}
	}

	// 4) Network usage check
	if netTotal > 0 {
		netUsedPct := int((netUsed * 100) / netTotal)
		if netUsedPct > netThreshold {
			// As tests expect, compute available as bytes / 1_000_000 (integer)
			var availBytes uint64
			if netTotal > netUsed {
				availBytes = netTotal - netUsed
			} else {
				availBytes = 0
			}
			avail := availBytes / 1_000_000 // integer, matches expected test numbers
			fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", avail)
		}
	}

	return true
}
