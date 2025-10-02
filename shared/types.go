package shared

type Game struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Price     float64       `json:"price"`
	Rating    float64       `json:"rating"`
	AgeRating EnumAgeRating `json:"age_rating"`
}
