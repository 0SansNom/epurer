package scantask

import (
	"path/filepath"
	"testing"

	"github.com/0SansNom/epurer/internal/ignorelist"
)

func TestIgnoredPredicate(t *testing.T) {
	root := t.TempDir()
	skipped := filepath.Join(root, "skip")
	kept := filepath.Join(root, "keep")

	list := &ignorelist.List{Paths: []string{skipped}}

	pred := Ignored{List: list}
	if !pred.Exclude(Entry{Path: filepath.Join(skipped, "nested"), Name: "nested"}) {
		t.Fatal("Ignored should drop a path under an ignored directory")
	}
	if pred.Exclude(Entry{Path: kept, Name: "keep"}) {
		t.Fatal("Ignored should keep a path outside the list")
	}
	if (Ignored{}).Exclude(Entry{Path: skipped, Name: "skip"}) {
		t.Fatal("Ignored without a list should keep the entry")
	}
}
