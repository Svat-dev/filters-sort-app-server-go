package shared

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// NormalizeSlice удаляет пустые элементы из среза.
func NormalizeSlice(s []string) []string {
	if len(s) == 0 {
		return s
	}
	result := make([]string, 0, len(s))
	for _, v := range s {
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

// GetEnv загружает переменные окружения из .env файла.
func GetEnv() (string, string, string) {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Ошибка загрузки .env файла")
	}

	dbURL := os.Getenv("DATABASE_URL")
	port := os.Getenv("PORT")
	clientUrl := os.Getenv("CLIENT_URL")

	if dbURL == "" || port == "" || clientUrl == "" {
		log.Fatal("Отсутствуют обязательные переменные окружения")
	}

	return dbURL, port, clientUrl
}

// BuildFilterWhereClause строит WHERE-часть запроса с параметризованными значениями.
func BuildFilterWhereClause(genres []string, platform, isAdultOnly string, lengthProp int) strings.Builder {
	var whereQuery strings.Builder
	var length int = lengthProp

	whereQuery.WriteString(` WHERE price > $1 AND price < $2 AND rating >= $3 AND (title ILIKE '%' || $4 || '%' OR developer ILIKE '%' || $4 || '%' OR publisher ILIKE '%' || $4 || '%')`)

	if isAdultOnly == "true" {
		whereQuery.WriteString(" AND age_rating IN ('AO', 'M')")
	}

	if platform != "" {
		whereQuery.WriteString(" AND $" + strconv.Itoa(length+1) + " = ANY(platforms)")
		length++
	}

	if len(genres) > 0 {
		whereQuery.WriteString(" AND genres @> $" + strconv.Itoa(length+1))
	}

	return whereQuery
}
