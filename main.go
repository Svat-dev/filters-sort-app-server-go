package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"main/shared"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func getEnv() (string, string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	dbURL := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")

	return dbURL, port
}

func main() {
	dbURL, port := getEnv()

	// Подключение к БД
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	log.Println("Подключение к БД выполнено успешно")

	http.HandleFunc("/games", func(w http.ResponseWriter, r *http.Request) {
		queryParams, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			http.Error(w, "Ошибка парсинга параметров", http.StatusBadRequest)
			return
		}

		sortType := queryParams.Get("sort")
		searchTerm := queryParams.Get("searchTerm")
		if searchTerm == "" {
			searchTerm = "%"
		}
		genres := strings.Split(queryParams.Get("genres"), ",")
		platforms := strings.Split(queryParams.Get("platforms"), ",")
		rating := queryParams.Get("rating")
		if rating == "" {
			rating = "0.0"
		}
		minPrice := queryParams.Get("minPrice")
		if minPrice == "" {
			minPrice = "0"
		}
		maxPrice := queryParams.Get("maxPrice")
		if maxPrice == "" {
			maxPrice = "100"
		}
		// isAdultOnly := queryParams.Get("isAdultOnly") == "true"

		var games []shared.Game

		// Выполнение запроса
		rows, err := conn.Query(context.Background(), "SELECT id, title, image, price, rating, age_rating, release_date, developer, publisher, genres, platforms FROM game WHERE price > $1 AND price < $2 AND rating >= $3 AND (title ILIKE '%' || $4 || '%' OR developer ILIKE '%' || $4 || '%' OR publisher ILIKE '%' || $4 || '%') AND genres @> $5 AND platforms @> $6", minPrice, maxPrice, rating, searchTerm, genres, platforms)
		if err != nil {
			http.Error(w, "Ошибка запроса", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var game shared.Game

			if err := rows.Scan(&game.ID, &game.Title, &game.ImageUrl, &game.Price, &game.Rating, &game.AgeRating, &game.ReleaseDate, &game.Developer, &game.Publisher, &game.Genres, &game.Platforms); err != nil {
				http.Error(w, "Ошибка сканирования", http.StatusInternalServerError)
				return
			}

			games = append(games, game)
		}

		switch sortType {
		case "HIGH_PRICE":
			sort.Sort(sort.Reverse(shared.SortByPrice(games)))
		case "LOW_PRICE":
			sort.Sort(shared.SortByPrice(games))
		case "OLDEST":
			sort.Sort(shared.SortByReleaseDate(games))
		default:
			sort.Sort(sort.Reverse(shared.SortByReleaseDate(games)))
		}

		// Отправка данных в JSON
		w.Header().Set("Content-Type", "application/json")

		data, err := json.Marshal(games)
		if err != nil {
			http.Error(w, "Ошибка сериализации", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	// Запуск сервера
	log.Printf("Сервер запущен на http://localhost:%s", port)
	http.ListenAndServe(":"+port, nil)
}
