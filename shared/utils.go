package shared

type SortByPrice []Game

func (a SortByPrice) Len() int           { return len(a) }
func (a SortByPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByPrice) Less(i, j int) bool { return a[i].Price < a[j].Price }

type SortByReleaseDate []Game

func (a SortByReleaseDate) Len() int           { return len(a) }
func (a SortByReleaseDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByReleaseDate) Less(i, j int) bool { return a[i].ReleaseDate.Before(a[j].ReleaseDate) }
