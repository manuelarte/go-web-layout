// Package pagination provides pagination utilities.
package pagination

import (
	"errors"
	"math"
)

var (
	ErrPageMustBeGreaterOrEqualThanZero = errors.New("page must be greater or equal than 0")
	ErrSizeMustBeGreaterOrEqualThanZero = errors.New("size must be greater or equal than 0")
)

type (

	// PageRequest represents a page request.
	PageRequest struct {
		page int
		size int
	}

	// Page output of a paginated query.
	Page[T any] struct {
		content       []T
		pageRequest   PageRequest
		totalElements int64
	}
)

func NewPageRequest(page, size int) (PageRequest, error) {
	if page < 0 {
		return PageRequest{}, ErrPageMustBeGreaterOrEqualThanZero
	}

	if size < 0 {
		return PageRequest{}, ErrSizeMustBeGreaterOrEqualThanZero
	}

	return PageRequest{
		page: page,
		size: size,
	}, nil
}

func MustPageRequest(page, size int) PageRequest {
	pr, err := NewPageRequest(page, size)
	if err != nil {
		panic(err)
	}

	return pr
}

func (pr PageRequest) Page() int {
	return pr.page
}

func (pr PageRequest) Size() int {
	return pr.size
}

func (pr PageRequest) Offset() int {
	return pr.page * pr.size
}

func NewPage[T any](content []T, pr PageRequest, totalElements int64) (Page[T], error) {
	if totalElements < 0 {
		return Page[T]{}, errors.New("total elements must be greater or equal than 0")
	}

	return Page[T]{
		content:       content,
		pageRequest:   pr,
		totalElements: totalElements,
	}, nil
}

func MustPage[T any](content []T, pr PageRequest, totalElements int64) Page[T] {
	page, err := NewPage(content, pr, totalElements)
	if err != nil {
		panic(err)
	}

	return page
}

func (p Page[T]) Content() []T {
	return p.content
}

func (p Page[T]) Size() int {
	return p.pageRequest.Size()
}

func (p Page[T]) Number() int {
	return p.pageRequest.Page()
}

func (p Page[T]) TotalElements() int64 {
	return p.totalElements
}

func (p Page[T]) TotalPages() int {
	return int(math.Ceil(float64(p.totalElements) / float64(p.pageRequest.Size())))
}
