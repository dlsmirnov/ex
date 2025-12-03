package main

import (
    "fmt"
    "net/http"
    "strconv"
    "strings"
)

func main() {
    // Пример запроса и парсинга данных
    resp, err := http.Get("http://srv.msk01.gigacorp.local/_stats")
    if err != nil || resp.StatusCode != 200 {
        fmt.Println("Unable to fetch server statistic")
        return
    }
    defer resp.Body.Close()

    var body []byte
    body, err = io.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Unable to fetch server statistic")
        return
    }

    parts := strings.Split(strings.TrimSpace(string(body)), ",")
    if len(parts) != 7 {
        fmt.Println("Unable to fetch server statistic")
        return
    }

    // Пример проверки Load Average
    loadAvg, _ := strconv.Atoi(parts[0])
    if loadAvg > 30 {
        fmt.Printf("Load Average is too high: %d\n", loadAvg)
    }

    // Здесь аналогично можно добавить проверки памяти, диска и сети
}
