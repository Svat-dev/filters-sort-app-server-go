package shared

type HTTPMethod string
type EnumAgeRating string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	PATCH  HTTPMethod = "PATCH"
	DELETE HTTPMethod = "DELETE"
)

const (
	E       EnumAgeRating = "E"       // Everyone
	E10Plus EnumAgeRating = "E10Plus" // Everyone 10+
	T       EnumAgeRating = "T"       // Teen
	M       EnumAgeRating = "M"       // Mature 17+
	AO      EnumAgeRating = "AO"      // Adults Only 18+
)
