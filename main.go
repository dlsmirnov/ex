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
    loadThreshold  = 30         // >30 -> warning
    memThresholdPct = 80        // >80%
    diskThresholdNum = 9        // used*10 > total*9  -> >90%
    netThresholdNum  = 9        // used*10 > total*9  -> >90%
    pollInterval   = 1 * time.Second
    maxErrors      = 3
)

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
            resp.Body.Close()
            errorCount++
            if errorCount >= maxErrors {
                fmt.Println("Unable to fetch server statistic")
            }
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

        parts := strings.Split(strings.TrimSpace(string(body)), ",")
        if len(parts) != 7 {
            errorCount++
            if errorCount >= maxErrors {
                fmt.Println("Unable to fetch server statistic")
            }
            time.Sleep(pollInterval)
            continue
        }

        // парсинг в int64
        vals := make([]int64, 7)
        parseErr := false
        for i := 0; i < 7; i++ {
            v, err := strconv.ParseInt(strings.TrimSpace(parts[i]), 10, 64)
            if err != nil {
                parseErr = true
                break
            }
            vals[i] = v
        }
        if parseErr {
            errorCount++
            if errorCount >= maxErrors {
                fmt.Println("Unable to fetch server statistic")
            }
            time.Sleep(pollInterval)
            continue
        }

        // успешный разбор -> сбрасываем счётчик ошибок
        errorCount = 0

        load := vals[0]
        memTotal := vals[1]
        memUsed := vals[2]
        diskTotal := vals[3]
        diskUsed := vals[4]
        netTotal := vals[5]
        netUsed := vals[6]

        // Load Average
        if load > loadThreshold {
            fmt.Printf("Load Average is too high: %d\n", load)
        }

        // Memory percent (целочисленно)
        if memTotal > 0 {
            memPct := memUsed * 100 / memTotal
            if memPct > memThresholdPct {
                fmt.Printf("Memory usage too high: %d%%\n", memPct)
            }
        }

        // Disk: check >90% used -> report free MB left
        if diskTotal > 0 {
            if diskUsed*10 > diskTotal*9 {
                freeBytes := diskTotal - diskUsed
                freeMB := freeBytes / 1024 / 1024
                fmt.Printf("Free disk space is too low: %d Mb left\n", freeMB)
            }
        }

        // Network: check >90% used -> report available Mbit/s
        if netTotal > 0 {
            if netUsed*10 > netTotal*9 {
                availBytes := netTotal - netUsed
                // Исправленный расчет с промежуточным делением
                availMbit := (availBytes / 1024 * 8)