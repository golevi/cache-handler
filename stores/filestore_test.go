package stores

import (
	"testing"
)

func TestPath(t *testing.T) {
	key := "abcdefg"
	fs := NewFileStore()

	out := fs.path(key)
	if out == "" {
		t.Error(out)
	}
}
