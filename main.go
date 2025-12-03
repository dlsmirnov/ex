package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultURL       = "http://srv.msk01.gigacorp.local/_stats"
	loadThreshold    = 30          // если load > 30
	memoryThresholdP = 80          // в процентах (80%)
	diskThresholdP   = 90          // в процентах (90%)
	networkThresholdP = 90         // в процентах (90%)
	checkInterval     = 1 * time.Second // интервал опроса
	maxErrors         = 3
)

func main() {
	// URL можно переопределить через переменную окружения STATS_URL (тесты её используют)
	statsURL := os.Getenv("STATS_URL")
	if statsURL == "" {
		statsURL = defaultURL
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	errorCount := 0

	for {
		// context с таймаутом на случай подвисания
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, statsURL, nil)
		if err != nil {
			cancel()
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		resp, err := client.Do(req)
		cancel()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// Разбираем тело: 7 чисел, разделённых запятыми
		parts := strings.Split(strings.TrimSpace(string(body)), ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// Парсим все как целые (int64)
		load, err0 := strconv.ParseInt(strings.TrimSpace(parts[0]), 10, 64)
		memTotal, err1 := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		memUsed, err2 := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
		diskTotal, err3 := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		diskUsed, err4 := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		netTotal, err5 := strconv.ParseInt(strings.TrimSpace(parts[5]), 10, 64)
		netUsed, err6 := strconv.ParseInt(strings.TrimSpace(parts[6]), 10, 64)

		if err0 != nil || err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// успешный разбор — сбрасываем счётчик ошибок
		errorCount = 0

		// 1) Load Average
		if load > loadThreshold {
			// печатаем целое значение (без десятичных)
			fmt.Printf("Load Average is too high: %d\n", load)
		}

		// 2) Memory usage in percent (целочисленно)
		if memTotal > 0 {
			memPercent := (memUsed * 100) / memTotal // целочисленное (floor)
			if memPercent > memoryThresholdP {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk usage: свободное пространство в MB
		if diskTotal > 0 {
			diskUsedPercent := (diskUsed * 100) / diskTotal
			if diskUsedPercent > diskThresholdP {
				freeBytes := diskTotal - diskUsed
				freeMB := freeBytes / (1024 * 1024) // целые мегабайты
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
			}
		}

		// 4) Network: свободная пропускная способность в Mbit/s
		if netTotal > 0 {
			netUsedPercent := (netUsed * 100) / netTotal
			if netUsedPercent > networkThresholdP {
				availBytes := netTotal - netUsed
				// переводим в Mbit/s: (bytes * 8) / (1024*1024)
				availMbit := (availBytes * 8) / (1024 * 1024)
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availMbit)
			}
		}

		// Ждём интервал
		time.Sleep(checkInterval)
	}
}
