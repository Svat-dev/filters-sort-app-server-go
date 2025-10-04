package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"slices"

	"strconv"
	"strings"

	"main/cerror"
	"main/shared"

	"github.com/asaskevich/govalidator"
	"github.com/jackc/pgx/v5"

	"github.com/rs/cors"
)

func main() {
	dbURL, port, clientUrl := shared.GetEnv()

	// Подключение к БД
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		panic(err)
	}
	defer conn.Close(context.Background())

	log.Println("Подключение к БД выполнено успешно")

	// Настройка обслуживания статических файлов из папки "uploads"
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	log.Println("Путь до статических файлов загружен")

	http.HandleFunc("/games", func(w http.ResponseWriter, r *http.Request) {
		queryParams, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			cerror.ThrowError(w, "Ошибка парсинга параметров", http.StatusBadRequest)
			return
		}

		// Вспомогательная функция для получения и проверки числовых параметров
		getIntParam := func(name string, defaultValue int) (int, error) {
			val := queryParams.Get(name)
			if val == "" {
				return defaultValue, nil
			}
			return strconv.Atoi(val)
		}

		page, err := getIntParam("page", 1)
		if err != nil || page < 1 {
			cerror.ThrowError(w, "Неверный номер страницы", http.StatusBadRequest)
			return
		}

		perPage, err := getIntParam("perPage", 30)
		if err != nil || perPage < 1 || perPage > 100 {
			cerror.ThrowError(w, "Неверное количество элементов на странице", http.StatusBadRequest)
			return
		}

		sortType := queryParams.Get("sort")
		validSortTypes := []string{"HIGH_PRICE", "LOW_PRICE", "OLDEST", "NEWEST", ""}
		if !slices.Contains(validSortTypes, sortType) {
			cerror.ThrowError(w, "Неверный тип сортировки", http.StatusBadRequest)
			return
		}

		searchTerm := queryParams.Get("searchTerm")
		if searchTerm == "" {
			searchTerm = "%"
		}

		genres := shared.NormalizeSlice(strings.Split(queryParams.Get("genres"), "|"))
		validGenres := []string{"Action", "Shooter", "Horror", "RPG", "Adventure", ""}
		for _, genre := range genres {
			if !slices.Contains(validGenres, genre) {
				cerror.ThrowError(w, "Неверный набор жанров", http.StatusBadRequest)
				return
			}
		}

		platform := queryParams.Get("platform")
		validPlatforms := []string{"PC", "Xbox", "PlayStation", "Nintendo", ""}
		if !slices.Contains(validPlatforms, platform) && platform != "" {
			cerror.ThrowError(w, "Неверная платформа", http.StatusBadRequest)
			return
		}

		rating := queryParams.Get("rating")
		if rating == "" {
			rating = "0.0"
		}
		if !govalidator.IsFloat(rating) {
			cerror.ThrowError(w, "Неверный формат рейтинга", http.StatusBadRequest)
			return
		}

		minPrice := queryParams.Get("minPrice")
		if minPrice == "" {
			minPrice = "0"
		}
		if !govalidator.IsFloat(minPrice) {
			cerror.ThrowError(w, "Неверный формат цены", http.StatusBadRequest)
			return
		}

		maxPrice := queryParams.Get("maxPrice")
		if maxPrice == "" {
			maxPrice = "100"
		}
		if !govalidator.IsFloat(maxPrice) {
			cerror.ThrowError(w, "Неверный формат цены", http.StatusBadRequest)
			return
		}

		isAdultOnly := queryParams.Get("isAdultOnly")
		if isAdultOnly != "" && isAdultOnly != "true" && isAdultOnly != "false" {
			cerror.ThrowError(w, `Неверный тип "только для взрослых"`, http.StatusBadRequest)
			return
		}

		var games []shared.Game
		var args []any

		var query strings.Builder
		var countQuery strings.Builder

		args = append(args, minPrice, maxPrice, rating, searchTerm)
		whereQuery := shared.BuildFilterWhereClause(genres, platform, isAdultOnly, len(args))

		if platform != "" {
			args = append(args, platform)
		}

		if len(genres) > 0 {
			args = append(args, genres)
		}

		baseQuery := `
			SELECT id, title, image, price, rating, age_rating, release_date, developer, publisher, genres, platforms 
			FROM game 
		`

		countBaseQuery := `
			SELECT COUNT(*) AS filtered_count 
			FROM game 
		`

		query.WriteString(baseQuery)
		countQuery.WriteString(countBaseQuery)

		if whereQuery.String() != "" {
			query.WriteString(whereQuery.String())
			countQuery.WriteString(whereQuery.String())
		}

		switch sortType {
		case "HIGH_PRICE":
			query.WriteString(" ORDER BY price DESC")
		case "LOW_PRICE":
			query.WriteString(" ORDER BY price ASC")
		case "OLDEST":
			query.WriteString(" ORDER BY release_date ASC")
		default:
			query.WriteString(" ORDER BY release_date DESC")
		}

		query.WriteString(" LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2))

		var response shared.GetGamesResponse

		rows, err := conn.Query(context.Background(), query.String(), append(args, perPage, (page-1)*perPage)...)
		if err != nil {
			log.Printf("Ошибка запроса: %v", err)
			cerror.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
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
				cerror.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
				return
			}
			games = append(games, game)
		}

		counted, err := conn.Query(context.Background(), countQuery.String(), args...)
		if err != nil {
			log.Printf("Ошибка запроса подсчета: %v", err)
			cerror.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}
		defer counted.Close()

		for counted.Next() {
			if err := counted.Scan(&response.Length); err != nil {
				log.Printf("Ошибка сканирования подсчета: %v", err)
				cerror.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
				return
			}
		}

		if games == nil {
			response.Games = []shared.Game{}
		} else {
			response.Games = games
		}

		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(response)
		if err != nil {
			log.Printf("Ошибка сериализации: %v", err)
			cerror.ThrowError(w, "Ошибка сервера", http.StatusInternalServerError)
			return
		}

		w.Write(data)
	})

	// Настройка CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{clientUrl}, // Разрешенные домены
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true, // Разрешить куки/авторизацию
		MaxAge:           100,  // Кэширование preflight-запроса
	})

	log.Printf("Сервер запущен на http://localhost:%s", port)

	handler := c.Handler(http.DefaultServeMux)
	http.ListenAndServe(":"+port, handler)
}
