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

	"github.com/asaskevich/govalidator"
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
		if ok := govalidator.IsIn(sortType, "HIGH_PRICE", "LOW_PRICE", "OLDEST", "NEWEST", ""); !ok {
			http.Error(w, "Неверный тип сортировки", http.StatusBadRequest)
		}
		searchTerm := queryParams.Get("searchTerm")
		if searchTerm == "" {
			searchTerm = "%"
		}

		genres := shared.NormalizeSlice(strings.Split(queryParams.Get("genres"), ","))       // TODO validation
		platforms := shared.NormalizeSlice(strings.Split(queryParams.Get("platforms"), ",")) // TODO validation

		rating := queryParams.Get("rating")
		if rating == "" {
			rating = "0.0"
		}
		if ok := govalidator.IsFloat(rating); !ok {
			http.Error(w, "Неверный формат рейтинга", http.StatusBadRequest)
		}

		minPrice := queryParams.Get("minPrice")
		if minPrice == "" {
			minPrice = "0"
		}
		if ok := govalidator.IsFloat(rating); !ok {
			http.Error(w, "Неверный формат цены", http.StatusBadRequest)
		}

		maxPrice := queryParams.Get("maxPrice")
		if maxPrice == "" {
			maxPrice = "100"
		}
		if ok := govalidator.IsFloat(rating); !ok {
			http.Error(w, "Неверный формат цены", http.StatusBadRequest)
		}

		var games []shared.Game
		var query strings.Builder
		var args []any

		baseQuery := `
			SELECT id, title, image, price, rating, age_rating, release_date, developer, publisher, genres, platforms 
			FROM game 
			WHERE price > $1 AND price < $2 AND rating >= $3 
			  AND (title ILIKE '%' || $4 || '%' OR developer ILIKE '%' || $4 || '%' OR publisher ILIKE '%' || $4 || '%')
		` //сделать ORDER BY тут а не далее

		args = append(args, minPrice, maxPrice, rating, searchTerm)
		query.WriteString(baseQuery)

		if len(genres) > 0 {
			query.WriteString(" AND genres @> $5")
			args = append(args, genres)
		}

		if len(platforms) > 0 {
			if len(genres) > 0 {
				query.WriteString(" AND platforms @> $6")
			} else {
				query.WriteString(" AND platforms @> $5")
			}
			args = append(args, platforms)
		}

		rows, err := conn.Query(context.Background(), query.String(), args...)
		if err != nil {
			log.Printf("Ошибка запроса: %v", err)
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var game shared.Game
			if err := rows.Scan(
				&game.ID,
				&game.Title,
				&game.ImageUrl,
				&game.Price,
				&game.Rating,
				&game.AgeRating,
				&game.ReleaseDate,
				&game.Developer,
				&game.Publisher,
				&game.Genres,
				&game.Platforms); err != nil {
				log.Printf("Ошибка сканирования: %v", err)
				http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
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

		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(games)
		if err != nil {
			log.Printf("Ошибка сериализации: %v", err)
			http.Error(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	log.Printf("Сервер запущен на http://localhost:%s", port)
	http.ListenAndServe(":"+port, nil)
}
