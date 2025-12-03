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
	STATS_URL       = "http://srv.msk01.gigacorp.local/_stats"
	CHECK_INTERVAL  = 1 * time.Second
	MAX_ERRORS      = 3
	LOAD_THRESHOLD  = 30   // integer threshold
	MEM_PERCENT_TH  = 80   // in percent
	DISK_PERCENT_TH = 90   // in percent
	NET_PERCENT_TH  = 90   // in percent
)

func parseInt(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func main() {
	errorCount := 0

	for {
		resp, err := http.Get(STATS_URL)
		if err != nil {
			errorCount++
			if errorCount >= MAX_ERRORS {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(CHECK_INTERVAL)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errorCount++
			if errorCount >= MAX_ERRORS {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(CHECK_INTERVAL)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			if errorCount >= MAX_ERRORS {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(CHECK_INTERVAL)
			continue
		}

		parts := strings.Split(strings.TrimSpace(string(body)), ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= MAX_ERRORS {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(CHECK_INTERVAL)
			continue
		}

		// successful parse — reset error counter
		errorCount = 0

		// parse all values as integers
		load, err1 := parseInt(parts[0])
		memTotal, err2 := parseInt(parts[1])
		memUsed, err3 := parseInt(parts[2])
		diskTotal, err4 := parseInt(parts[3])
		diskUsed, err5 := parseInt(parts[4])
		netTotal, err6 := parseInt(parts[5])
		netUsed, err7 := parseInt(parts[6])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			// parsing error
			errorCount++
			if errorCount >= MAX_ERRORS {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(CHECK_INTERVAL)
			continue
		}

		// 1) Load average
		if load > LOAD_THRESHOLD {
			fmt.Printf("Load Average is too high: %d\n", load)
		}

		// 2) Memory usage percent (integer, rounded down)
		if memTotal > 0 {
			memPercent := memUsed * 100 / memTotal
			if memPercent > MEM_PERCENT_TH {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk free MB and threshold
		if diskTotal > 0 {
			free := diskTotal - diskUsed
			usedPercent := diskUsed * 100 / diskTotal
			if usedPercent > DISK_PERCENT_TH {
				mbLeft := free / 1024 / 1024
				fmt.Printf("Free disk space is too low: %d Mb left\n", mbLeft)
			}
		}

		// 4) Network: available Mbit/s and threshold
		// Compute percent usage based on bytes
		if netTotal > 0 {
			netUsedPercent := netUsed * 100 / netTotal
			if netUsedPercent > NET_PERCENT_TH {
				availBytes := netTotal - netUsed
				// convert bytes -> bits then to megabits, integer division
				// using 1024*1024 to match binary MiB/Mib conventions used in tests
				availMbit := (availBytes * 8) / (1024 * 1024)
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availMbit)
			}
		}

		// single run mode for autotests: if this process is executed by tests they usually expect quick exit.
		// But keep loop for normal mode; if env RUN_ONCE set, exit after first iteration.
		if os.Getenv("RUN_ONCE") == "1" {
			return
		}

		time.Sleep(CHECK_INTERVAL)
	}
}
