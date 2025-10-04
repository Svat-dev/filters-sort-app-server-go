package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"main/error"
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
			error.ThrowError(w, "Ошибка парсинга параметров", http.StatusBadRequest)
			return
		}

		page := queryParams.Get("page")
		if page == "" {
			page = "1"
		}
		page64, err := strconv.Atoi(page)
		if err != nil {
			error.ThrowError(w, "Неправильный тип числа", http.StatusBadRequest)
			return
		}

		perPage := queryParams.Get("perPage")
		if perPage == "" {
			perPage = "30"
		}
		perPage64, err := strconv.Atoi(perPage)
		if err != nil {
			error.ThrowError(w, "Неправильный тип числа", http.StatusBadRequest)
			return
		}

		sortType := queryParams.Get("sort")
		if ok := govalidator.IsIn(sortType, "HIGH_PRICE", "LOW_PRICE", "OLDEST", "NEWEST", ""); !ok {
			error.ThrowError(w, "Неверный тип сортировки", http.StatusBadRequest)
			return
		}

		searchTerm := queryParams.Get("searchTerm")
		if searchTerm == "" {
			searchTerm = "%"
		}

		genres := shared.NormalizeSlice(strings.Split(queryParams.Get("genres"), ","))
		for _, genre := range genres {
			if ok := govalidator.IsIn(genre, "Action", "Shooter", "Horror", "RPG", "Adventure"); !ok {
				error.ThrowError(w, "Неверный набор жанров", http.StatusBadRequest)
				return
			}
		}

		platforms := shared.NormalizeSlice(strings.Split(queryParams.Get("platforms"), ","))
		for _, platform := range platforms {
			if ok := govalidator.IsIn(platform, "PC", "Xbox", "PlayStation", "Nintendo"); !ok {
				error.ThrowError(w, "Неверный набор платформ", http.StatusBadRequest)
				return
			}
		}

		rating := queryParams.Get("rating")
		if rating == "" {
			rating = "0.0"
		}
		if ok := govalidator.IsFloat(rating); !ok {
			error.ThrowError(w, "Неверный формат рейтинга", http.StatusBadRequest)
			return
		}

		minPrice := queryParams.Get("minPrice")
		if minPrice == "" {
			minPrice = "0"
		}
		if ok := govalidator.IsFloat(minPrice); !ok {
			error.ThrowError(w, "Неверный формат цены", http.StatusBadRequest)
			return
		}

		maxPrice := queryParams.Get("maxPrice")
		if maxPrice == "" {
			maxPrice = "100"
		}
		if ok := govalidator.IsFloat(maxPrice); !ok {
			error.ThrowError(w, "Неверный формат цены", http.StatusBadRequest)
			return
		}

		isAdultOnly := queryParams.Get("isAdultOnly")
		if ok := govalidator.IsIn(isAdultOnly, "true", "false", ""); !ok {
			error.ThrowError(w, `Неверный тип "только для взрослых"`, http.StatusBadRequest)
		}

		var games []shared.Game
		var query strings.Builder
		var args []any

		baseQuery := `
			SELECT id, title, image, price, rating, age_rating, release_date, developer, publisher, genres, platforms 
			FROM game 
			WHERE price > $1 AND price < $2 AND rating >= $3 
			  AND (title ILIKE '%' || $4 || '%' OR developer ILIKE '%' || $4 || '%' OR publisher ILIKE '%' || $4 || '%')
		`

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

		switch sortType {
		case "HIGH_PRICE":
			query.WriteString(" ORDER BY price desc")
		case "LOW_PRICE":
			query.WriteString(" ORDER BY price asc")
		case "OLDEST":
			query.WriteString(" ORDER BY release_date asc")
		default:
			query.WriteString(" ORDER BY release_date desc")
		}

		query.WriteString(" LIMIT " + strconv.Itoa(perPage64) + " OFFSET " + strconv.Itoa((page64-1)*perPage64))

		rows, err := conn.Query(context.Background(), query.String(), args...)
		if err != nil {
			log.Printf("Ошибка запроса: %v", err)
			error.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
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
				error.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
				return
			}
			games = append(games, game)
		}

		var filteredGames []shared.Game

		for _, game := range games {
			if isAdultOnly != "true" {
				filteredGames = append(filteredGames, game)
			}

			if govalidator.IsIn(string(game.AgeRating), string(shared.M), string(shared.AO)) {
				filteredGames = append(filteredGames, game)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(filteredGames)
		if err != nil {
			log.Printf("Ошибка сериализации: %v", err)
			error.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	log.Printf("Сервер запущен на http://localhost:%s", port)
	http.ListenAndServe(":"+port, nil)
}
