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
	defaultURL   = "http://srv.msk01.gigacorp.local/_stats"
	checkInterval = 1 * time.Second
	maxErrors     = 3
)

func getenvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func toInt64(s string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(s), 10, 64)
}

func main() {
	url := getenvOrDefault("STATS_URL", defaultURL)
	errorCount := 0

	for {
		resp, err := http.Get(url)
		if err != nil {
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

		if resp.StatusCode != http.StatusOK {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		text := strings.TrimSpace(string(body))
		parts := strings.Split(text, ",")
		if len(parts) != 7 {
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// Успешный разбор — сбрасываем счётчик ошибок
		errorCount = 0

		// Парсим все значения как целые (байты, байт/с и т.п.)
		loadAvg, err1 := toInt64(parts[0])
		memTotal, err2 := toInt64(parts[1])
		memUsed, err3 := toInt64(parts[2])
		diskTotal, err4 := toInt64(parts[3])
		diskUsed, err5 := toInt64(parts[4])
		netTotal, err6 := toInt64(parts[5])
		netUsed, err7 := toInt64(parts[6])

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil || err7 != nil {
			// если хоть одно значение не распарсилось — считаем как ошибка получения данных
			errorCount++
			if errorCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
			time.Sleep(checkInterval)
			continue
		}

		// 1) Load Average
		if loadAvg > 30 {
			fmt.Printf("Load Average is too high: %d\n", loadAvg)
		}

		// 2) Memory usage: проценты — целая часть (вниз)
		if memTotal > 0 {
			memPercent := (memUsed * 100) / memTotal // integer division, floor
			if memPercent > 80 {
				fmt.Printf("Memory usage too high: %d%%\n", memPercent)
			}
		}

		// 3) Disk usage: если used > 90% => сообщаем сколько MB свободно (целые мегабайты)
		if diskTotal > 0 {
			if diskUsed*10 > diskTotal*9 { // diskUsed/diskTotal > 0.9
				freeBytes := diskTotal - diskUsed
				freeMB := freeBytes / (1024 * 1024) // целые мегабайты (floor)
				fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
			}
		}

		// 4) Network: если used > 90% => сообщаем свободную пропускную в Mbit/s (целая часть)
		if netTotal > 0 {
			if netUsed*10 > netTotal*9 { // netUsed/netTotal > 0.9
				availableBytesPerSec := netTotal - netUsed
				// перевод в Mbit/s: bytes/sec * 8 / 1_000_000, целая часть
				availableMbit := (availableBytesPerSec * 8) / 1000000
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availableMbit)
			}
		}

		time.Sleep(checkInterval)
	}
}
