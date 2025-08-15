package pagination

type Page[T any] struct {
	Data          []T
	Size          int
	TotalElements int64
	TotalPages    int
	Number        int
}
