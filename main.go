package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

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

	// Маршрут /data
	http.HandleFunc("/data", func(w http.ResponseWriter, r *http.Request) {
		var games []shared.Game

		// Выполнение запроса
		rows, err := conn.Query(context.Background(), "SELECT id, title, price, rating, age_rating FROM game")
		if err != nil {
			http.Error(w, "Ошибка запроса", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var game shared.Game

			if err := rows.Scan(&game.ID, &game.Title, &game.Price, &game.Rating, &game.AgeRating); err != nil {
				http.Error(w, "Ошибка сканирования", http.StatusInternalServerError)
				return
			}

			games = append(games, game)
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
