package scantask

import "github.com/0SansNom/epurer/internal/ignorelist"

// Entry is one filesystem item a predicate can accept or reject.
type Entry struct {
	Path string
	Name string
	Size int64
}

// Predicate decides whether an entry is dropped from a scan.
// Exclude reports true when the entry must not become a clean target.
type Predicate interface {
	Exclude(Entry) bool
}

// Ignored drops an entry covered by the persistent ignore list.
type Ignored struct {
	List *ignorelist.List
}

func (i Ignored) Exclude(e Entry) bool {
	if i.List == nil {
		return false
	}
	return i.List.IsIgnored(e.Path)
}
