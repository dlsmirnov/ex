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
	loadThreshold  = 30      // если Load Average > 30
	memThresholdP  = 80      // если память > 80%
	diskThresholdP = 90      // если диск > 90%
	netThresholdP  = 90      // если сеть > 90%
	checkInterval  = 5 * time.Second
	maxErrors      = 3
)

func main() {
	// Позволяет переопределить URL через переменную окружения (удобно для локального тестирования)
	url := statsURL
	if v := os.Getenv("STATS_URL"); v != "" {
		url = v
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	errCount := 0

	for {
		ok := fetchAndCheck(client, url)
		if ok {
			// успешный запрос — сбрасываем ошибки
			errCount = 0
		} else {
			errCount++
			if errCount >= maxErrors {
				fmt.Println("Unable to fetch server statistic")
			}
		}
		time.Sleep(checkInterval)
	}
}

func fetchAndCheck(client *http.Client, url string) bool {
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false
	}

	fields := strings.Split(strings.TrimSpace(string(body)), ",")
	if len(fields) != 7 {
		return false
	}

	// Парсим все поля как 64-bit целые (байты / значения)
	load, ok1 := parseInt(fields[0])
	memTotal, ok2 := parseUint64(fields[1])
	memUsed, ok3 := parseUint64(fields[2])
	diskTotal, ok4 := parseUint64(fields[3])
	diskUsed, ok5 := parseUint64(fields[4])
	netTotal, ok6 := parseUint64(fields[5])
	netUsed, ok7 := parseUint64(fields[6])

	if !(ok1 && ok2 && ok3 && ok4 && ok5 && ok6 && ok7) {
		return false
	}

	// 1) Load Average
	if load > loadThreshold {
		// вывод без десятичных частей
		fmt.Printf("Load Average is too high: %d\n", load)
	}

	// 2) Memory usage % (целая часть процента)
	// Проверка >80% делаем через целые числа, чтобы не иметь погрешностей
	if memTotal > 0 {
		memPerc := (memUsed * 100) / memTotal // целая часть
		if memPerc > memThresholdP {
			fmt.Printf("Memory usage too high: %d%%\n", memPerc)
		}
	}

	// 3) Disk free MB left
	if diskTotal > 0 {
		diskUsedPercent := (diskUsed * 100) / diskTotal
		if diskUsedPercent > diskThresholdP {
			freeBytes := diskTotal - diskUsed
			freeMB := freeBytes / 1024 / 1024
			fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
		}
	}

	// 4) Network available - считаем в Mbit/s (целая часть)
	if netTotal > 0 {
		netUsedPercent := (netUsed * 100) / netTotal
		if netUsedPercent > netThresholdP {
			availBytes := netTotal - netUsed
			// перевод в мегабиты в секунду: bytes -> bits (*)8, делим на 1024*1024 и берем целую часть
			availMbit := (availBytes * 8) / 1024 / 1024
			fmt.Printf("Network bandwidth usage high: %d Mbit/s available\n", availMbit)
		}
	}

	return true
}

func parseInt(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		// иногда load average может быть дробным — попытаемся отбросить дробную часть
		if strings.Contains(s, ".") {
			parts := strings.SplitN(s, ".", 2)
			v2, err2 := strconv.ParseInt(parts[0], 10, 64)
			if err2 == nil {
				return v2, true
			}
		}
		return 0, false
	}
	return v, true
}

func parseUint64(s string) (uint64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		// иногда числа в тестах большие — но ParseUint справится; если не, возвращаем false
		return 0, false
	}
	return v, true
}
