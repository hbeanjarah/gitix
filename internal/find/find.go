package find

import (
	"sort"
	"strings"

	"github.com/hbeanjarah/gitix/internal/git"
)

// Filter returns the branches whose name contains term (case-insensitive),
// most recent commit first. An empty term matches every branch.
func Filter(branches []git.Branch, term string) []git.Branch {
	term = strings.ToLower(term)
	var out []git.Branch
	for _, b := range branches {
		if strings.Contains(strings.ToLower(b.Name), term) {
			out = append(out, b)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].CommitTime.After(out[j].CommitTime)
	})
	return out
}
