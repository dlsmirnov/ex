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
	statsURL       = "http://srv.msk01.gigacorp.local/_stats" // URL сервера (в автотестах этот хост подменяется)
	loadThreshold  = 30   // если LoadAverage > 30 -> сообщение
	memPercentThr  = 80   // если used_mem*100/total_mem > 80 -> сообщение
	diskPercentThr = 90   // если used_disk*100/total_disk > 90 -> сообщение
	netPercentThr  = 90   // если used_net*100/total_net > 90 -> сообщение

	maxErrors    = 3
	pollInterval = 1 * time.Second
	httpTimeout  = 5 * time.Second
)

func main() {
	// Если в окружении задано STATS_URL (для локального тестирования), используем его.
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

		// Успешный разбор — сбрасываем счетчик ошибок и флаг
		errorCount = 0
		printedUnable = false

		// парсим значения как uint64 (безопасно для больших чисел)
		loadAvg, ok := parseUint(parts[0])
		if !ok { time.Sleep(pollInterval); continue }
		memTotal, ok := parseUint(parts[1])
		if !ok { time.Sleep(pollInterval); continue }
		memUsed, ok := parseUint(parts[2])
		if !ok { time.Sleep(pollInterval); continue }
		diskTotal, ok := parseUint(parts[3])
		if !ok { time.Sleep(pollInterval); continue }
		diskUsed, ok := parseUint(parts[4])
		if !ok { time.Sleep(pollInterval); continue }
		netTotal, ok := parseUint(parts[5])
		if !ok { time.Sleep(pollInterval); continue }
		netUsed, ok := parseUint(parts[6])
		if !ok { time.Sleep(pollInterval); continue }

		// Проверки. ВНИМАНИЕ: используем целочисленную арифметику и точный формат вывода.
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

		// 3) Disk free in MB and disk percent
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
				// перевод в Mbit/s, как в автотестах ожидается: divide by 1_000_000
				availableMbit := availableBytes / 1000000
				fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availableMbit)
			}
		}

		// ждем перед следующим опросом
		time.Sleep(pollInterval)
	}
}

// parseUint парсит строку в uint64, возвращает (value, ok)
func parseUint(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	if v, err := strconv.ParseUint(s, 10, 64); err == nil {
		return v, true
	}
	// fallback: если вдруг отрицательное или signed, пробуем ParseInt и приводим
	if v2, err2 := strconv.ParseInt(s, 10, 64); err2 == nil && v2 >= 0 {
		return uint64(v2), true
	}
	return 0, false
}


