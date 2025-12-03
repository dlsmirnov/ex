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
	statsURL         = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold    = 30.0
	memoryThreshold  = 0.8
	diskThreshold    = 0.9
	networkThreshold = 0.9
	pollInterval     = 1 * time.Second
	maxErrors        = 3
)

func toFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	return strconv.ParseFloat(s, 64)
}

func main() {
	errorCount := 0

	for {
		resp, err := http.Get(statsURL)
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(pollInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			resp.Body.Close()
			time.Sleep(pollInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(pollInterval)
			continue
		}

		text := strings.TrimSpace(string(body))
		parts := strings.Split(text, ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(pollInterval)
			continue
		}

		// успешный разбор — сбрасываем счётчик ошибок
		errorCount = 0

		// Парсинг полей
		loadAvg, err1 := toFloat(parts[0])
		memTotal, err2 := toFloat(parts[1])
		memUsed, err3 := toFloat(parts[2])
		diskTotal, err4 := toFloat(parts[3])
		diskUsed, err5 := toFloat(parts[4])
		netTotal, err6 := toFloat(parts[5])
		netUsed, err7 := toFloat(parts[6])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			// формат неверный
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(pollInterval)
			continue
		}

		// Load Average
		if loadAvg > loadThreshold {
			// вывод как целое число, без дробной части
			fmt.Printf("Load Average is too high: %.0f\n", loadAvg)
		}

		// Memory usage percent
		if memTotal > 0 {
			memUsage := memUsed / memTotal
			if memUsage > memoryThreshold {
				percent := memUsage * 100
				fmt.Printf("Memory usage too high: %.0f%%\n", percent)
			}
		}

		// Disk free space in MB left
		if diskTotal > 0 {
			if diskUsed/diskTotal > diskThreshold {
				freeBytes := diskTotal - diskUsed
				freeMb := freeBytes / 1024 / 1024
				fmt.Printf("Free disk space is too low: %.0f Mb left\n", freeMb)
			}
		}

		// Network available in Mbit/s
		if netTotal > 0 {
			if netUsed/netTotal > networkThreshold {
				availableBytes := netTotal - netUsed
				availableMbit := (availableBytes * 8) / (1024 * 1024)
				fmt.Printf("Network bandwidth usage high: %.0f Mbit/s available\n", availableMbit)
			}
		}

		// Ждём немного и повторяем — тесты в CI имитируют сценарии и останавливают процесс сами.
		time.Sleep(pollInterval)
	}
}
