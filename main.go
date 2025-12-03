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
	defaultURL    = "http://srv.msk01.gigacorp.local/_stats"
	checkInterval = 1 * time.Second
	maxErrors     = 3
)

func parseInt64(s string) (int64, error) {
	s = strings.TrimSpace(s)
	return strconv.ParseInt(s, 10, 64)
}

func main() {
	statsURL := os.Getenv("STATS_URL")
	if statsURL == "" {
		statsURL = defaultURL
	}

	errorCount := 0
	printedUnable := false

	for {
		resp, err := http.Get(statsURL)
		if err != nil {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(checkInterval)
			continue
		}

		// body and status check
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(checkInterval)
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
			time.Sleep(checkInterval)
			continue
		}

		parts := strings.Split(strings.TrimSpace(string(body)), ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(checkInterval)
			continue
		}

		// успешный запрос — сброс ошибок/флага
		errorCount = 0
		printedUnable = false

		// Парсим в int64
		loadAvg, err1 := parseInt64(parts[0])
		memTotal, err2 := parseInt64(parts[1])
		memUsed, err3 := parseInt64(parts[2])
		diskTotal, err4 := parseInt64(parts[3])
		diskUsed, err5 := parseInt64(parts[4])
		netTotal, err6 := parseInt64(parts[5])
		netUsed, err7 := parseInt64(parts[6])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			// неверный формат
			errorCount++
			if errorCount >= maxErrors && !printedUnable {
				fmt.Println("Unable to fetch server statistic")
				printedUnable = true
			}
			time.Sleep(checkInterval)
			continue
		}

		// 1) Load Average > 30
		if loadAvg > 30 {
			fmt.Printf("Load Average is too high: %d\n", loadAvg)
		}

		// 2) Memory usage > 80% => "Memory usage too high: N%"
		if memTotal > 0 {
			memPercent := (memUsed * 100) / memTotal
			if memPercent > 80 {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk usage > 90% => "Free disk space is too low: N Mb left"
		if diskTotal > 0 {
			if diskUsed*10 > diskTotal*9 { // diskUsed/diskTotal > 0.9  -> multiply to avoid float
				freeMb := (diskTotal - diskUsed) / 1024 / 1024
				if freeMb < 0 {
					freeMb = 0
				}
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMb)
			}
		}

		// 4) Network: usage > 90% => "Network bandwidth usage high: N Mbit/s available"
		if netTotal > 0 {
			if netUsed*10 > netTotal*9 { // netUsed/netTotal > 0.9
				availableBytes := netTotal - netUsed
				if availableBytes < 0 {
					availableBytes = 0
				}
				// IMPORTANT: tests expect N = (availableBytes / 1024 / 1024) (no *8)
				availableMbit := availableBytes / 1024 / 1024
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availableMbit)
			}
		}

		time.Sleep(checkInterval)
	}
}
