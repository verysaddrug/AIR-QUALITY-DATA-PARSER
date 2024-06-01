package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"Final/db"
	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
	_ "github.com/lib/pq"
)

const maxParallelRequests = 1 // Максимальное количество параллельных запросов

func main() {
	// Запуск xvfb
	cmd := exec.Command("Xvfb", ":99", "-ac", "-screen", "0", "1280x1024x16")
	if err := cmd.Start(); err != nil {
		log.Fatalf("Ошибка запуска Xvfb: %v", err)
	}
	defer cmd.Process.Kill()

	// Установка переменной окружения для использования xvfb
	os.Setenv("DISPLAY", ":99")

	// Получение координат и временного интервала из аргументов командной строки
	latitude := flag.Float64("lat", 60.1695, "Latitude")
	longitude := flag.Float64("lon", 24.9354, "Longitude")
	startDate := flag.String("start", "2023-05-28", "Start Date (YYYY-MM-DD)")
	endDate := flag.String("end", "2024-05-30", "End Date (YYYY-MM-DD)")
	flag.Parse()

	// Парсим даты
	start, err := time.Parse("2006-01-02", *startDate)
	if err != nil {
		log.Fatalf("Ошибка парсинга startDate: %v", err)
	}
	end, err := time.Parse("2006-01-02", *endDate)
	if err != nil {
		log.Fatalf("Ошибка парсинга endDate: %v", err)
	}

	// Настройки базы данных из переменных окружения
	dbConfig := db.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     5432, // Значение по умолчанию
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Проверяем готовность базы данных
	if err := waitForDB(dbConfig, 30*time.Second); err != nil {
		log.Fatalf("База данных не готова: %v", err)
	}

	// Подключаемся к базе данных и создаем таблицы
	dbConn, err := db.SetupDB(dbConfig)
	if err != nil {
		log.Fatalf("Ошибка настройки базы данных: %v", err)
	}
	defer dbConn.Close()

	// Используем семафор для ограничения количества параллельных запросов
	sem := make(chan struct{}, maxParallelRequests)

	var wg sync.WaitGroup
	for t := start; t.Before(end); t = t.Add(time.Hour) {
		wg.Add(1)
		sem <- struct{}{} // Захватываем семафор
		go func(t time.Time) {
			defer wg.Done()
			defer func() { <-sem }() // Освобождаем семафор

			processTime(dbConn, t, *latitude, *longitude)
		}(t)
	}
	wg.Wait()
}

func waitForDB(config db.DBConfig, timeout time.Duration) error {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode)

	start := time.Now()
	for {
		db, err := sql.Open("postgres", connStr)
		if err != nil {
			return err
		}
		defer db.Close()

		err = db.Ping()
		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("время ожидания истекло: %v", err)
		}

		time.Sleep(1 * time.Second)
	}
}

func processTime(dbConn *sql.DB, t time.Time, latitude, longitude float64) {
	date := t.Format("2006/01/02")
	timeStr := t.Format("1504Z")

	// Сначала получаем данные для PM1
	urlPM1 := fmt.Sprintf("https://earth.nullschool.net/#%s/%s/particulates/surface/level/overlay=pm1/orthographic=%f,%f,631/loc=%f,%f",
		date, timeStr, longitude, latitude, longitude, latitude)
	coords, windInfo, pm1 := fetchData(urlPM1)
	if coords == "" || windInfo == "" || pm1 == "" {
		log.Printf("Не удалось получить данные для PM1 на %s", t)
		return
	}

	// Получаем данные для PM2.5
	urlPM25 := fmt.Sprintf("https://earth.nullschool.net/#%s/%s/particulates/surface/level/overlay=pm2.5/orthographic=%f,%f,631/loc=%f,%f",
		date, timeStr, longitude, latitude, longitude, latitude)
	_, _, pm25 := fetchData(urlPM25)
	if pm25 == "" {
		log.Printf("Не удалось получить данные для PM2.5 на %s", t)
		return
	}

	// Получаем данные для PM10
	urlPM10 := fmt.Sprintf("https://earth.nullschool.net/#%s/%s/particulates/surface/level/overlay=pm10/orthographic=%f,%f,631/loc=%f,%f",
		date, timeStr, longitude, latitude, longitude, latitude)
	_, _, pm10 := fetchData(urlPM10)
	if pm10 == "" {
		log.Printf("Не удалось получить данные для PM10 на %s", t)
		return
	}

	// Форматируем информацию о ветре
	direction, speed := extractWindData(windInfo)

	// Сохраняем данные в базу данных
	err := saveData(dbConn, t, coords, direction, speed, pm1, pm25, pm10)
	if err != nil {
		log.Fatalf("Ошибка сохранения данных: %v", err)
	}

	// Печатаем извлеченные данные для проверки
	fmt.Printf("Дата и время: %s\n", t)
	fmt.Printf("Координаты: %s\n", coords)
	fmt.Printf("Направление ветра: %d\n", direction)
	fmt.Printf("Скорость ветра: %d\n", speed)
	fmt.Printf("Концентрация PM1: %s\n", pm1)
	fmt.Printf("Концентрация PM2.5: %s\n", pm25)
	fmt.Printf("Концентрация PM10: %s\n", pm10)
}

// Функция для извлечения данных из HTML
func extractData(htmlContent, selector string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Fatalf("Ошибка парсинга HTML: %v", err)
	}
	return doc.Find(selector).Text()
}

// Функция для извлечения направления и скорости ветра
func extractWindData(windInfo string) (int, int) {
	//fmt.Println(windInfo)
	parts := strings.Fields(windInfo)
	var direction, speed int
	fmt.Sscanf(parts[0], "%d", &direction) //  направление - первое слово
	fmt.Sscanf(parts[2], "%d", &speed)     //  скорость - третье слово
	return direction, speed
}

// Функция для сохранения данных в базу данных
func saveData(db *sql.DB, timestamp time.Time, coords string, direction, speed int, pm1, pm25, pm10 string) error {
	query := `INSERT INTO air_quality_data (timestamp, coords, wind_direction, wind_speed, pm1, pm25, pm10) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := db.Exec(query, timestamp, coords, direction, speed, pm1, pm25, pm10)
	return err
}

// Функция для получения данных с сайта с повторными попытками
func fetchData(url string) (string, string, string) {
	const maxRetries = 3
	const retryDelay = 5 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Создаем контекст для chromedp
		ctx, cancel := chromedp.NewContext(context.Background())
		defer cancel()

		// Устанавливаем таймаут
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Переменная для хранения данных
		var htmlContent string

		// Запускаем задачи chromedp
		err := chromedp.Run(ctx,
			chromedp.Navigate(url),
			chromedp.Sleep(2*time.Second), // Ожидаем загрузки данных
			chromedp.OuterHTML("html", &htmlContent),
		)
		if err != nil {
			log.Printf("Попытка %d: ошибка загрузки страницы: %v", attempt, err)
			if attempt < maxRetries {
				log.Printf("Повторная попытка через %s...", retryDelay)
				time.Sleep(retryDelay)
				continue
			} else {
				log.Fatalf("Все попытки загрузки страницы исчерпаны: %v", err)
			}
		}

		// Теперь у нас есть полный HTML контент страницы, можно его парсить
		coords := extractData(htmlContent, `div[data-name="spotlight-coords"]`)
		windInfo := extractData(htmlContent, `div[data-name="spotlight-a"]`)
		pm := extractData(htmlContent, `div[data-name="spotlight-b"] div[aria-label]`)

		return coords, windInfo, pm
	}

	// Если все попытки не удались, возвращаем пустые строки
	return "", "", ""
}
