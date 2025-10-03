package shared

import "time"

type Game struct {
	ID string `json:"id"`

	Title    string `json:"title"`
	ImageUrl string `json:"image"`

	Developer string `json:"developer"`
	Publisher string `json:"publisher"`

	Genres    []string `json:"genres"`
	Platforms []string `json:"platforms"`

	Price     float64       `json:"price"`
	Rating    float64       `json:"rating"`
	AgeRating EnumAgeRating `json:"age_rating"`

	ReleaseDate time.Time `json:"release_date"`
}
