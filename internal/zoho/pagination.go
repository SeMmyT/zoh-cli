package zoho

// PageIterator provides generic offset-based pagination for Zoho APIs
type PageIterator[T any] struct {
	fetchFunc func(start, limit int) ([]T, error)
	pageSize  int
	current   int
	done      bool
}

// NewPageIterator creates a new PageIterator with the given fetch function and page size
func NewPageIterator[T any](fetchFunc func(start, limit int) ([]T, error), pageSize int) *PageIterator[T] {
	if pageSize <= 0 {
		pageSize = 50 // Default page size
	}
	return &PageIterator[T]{
		fetchFunc: fetchFunc,
		pageSize:  pageSize,
		current:   0,
		done:      false,
	}
}

// FetchAll fetches all pages until no more results are available
func (p *PageIterator[T]) FetchAll() ([]T, error) {
	var results []T

	for !p.done {
		page, err := p.FetchPage(p.current)
		if err != nil {
			return nil, err
		}

		results = append(results, page...)

		// If we got fewer results than page size, we're done
		if len(page) < p.pageSize {
			p.done = true
			break
		}

		p.current += len(page)
	}

	return results, nil
}

// FetchPage fetches a single page starting at the given offset
func (p *PageIterator[T]) FetchPage(start int) ([]T, error) {
	return p.fetchFunc(start, p.pageSize)
}
