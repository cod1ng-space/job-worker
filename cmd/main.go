package main

import (
	"flag"
	"fmt"
	"log"

	"job-worker/internal/api"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	// Конфигурация
	goroutines_count := flag.Int("workers_count", 5, "Количество воркеров")
	port := flag.Int("port", 8080, "Порт сервера")
	retries := flag.Int("retries", 3, "Количество повторных попыток")
	timeout := flag.Int("timeout", 3, "Ограничение времени на выполнение одной задачи(таймаут)")
	flag.Parse()

	h := api.NewHandler(*goroutines_count, *retries, *timeout)

	e := echo.New()
	e.Use(middleware.Recover())

	e.POST("/jobs", h.PostJob)   // Создание задачи
	e.PUT("/jobs/:id", h.PutJob) // Создание или обновление задачи
	e.GET("/jobs/:id", h.GetJob) // Получение информации о статусе задачи

	log.Printf("Сервер запущен на порту %d\n", *port)
	e.Logger.Fatal(e.Start(fmt.Sprintf(":%d", *port)))
}
