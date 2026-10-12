package find

import (
	"testing"
	"time"

	"github.com/hbeanjarah/gitix/internal/git"
)

func branch(name string, age time.Duration) git.Branch {
	return git.Branch{Name: name, CommitTime: time.Unix(1_700_000_000, 0).Add(-age)}
}

func names(bs []git.Branch) []string {
	var out []string
	for _, b := range bs {
		out = append(out, b.Name)
	}
	return out
}

func TestFilterIsCaseInsensitiveSubstring(t *testing.T) {
	bs := []git.Branch{branch("feat/trash-module", 0), branch("fix/login", 0), branch("fix/Trash-icon", 0)}
	got := names(Filter(bs, "TRASH"))
	if len(got) != 2 || got[0] != "feat/trash-module" || got[1] != "fix/Trash-icon" {
		t.Errorf("got %v", got)
	}
}

func TestFilterSortsMostRecentFirst(t *testing.T) {
	bs := []git.Branch{branch("old", 48*time.Hour), branch("new", time.Hour), branch("mid", 24*time.Hour)}
	got := names(Filter(bs, ""))
	if len(got) != 3 || got[0] != "new" || got[1] != "mid" || got[2] != "old" {
		t.Errorf("got %v", got)
	}
}

func TestFilterNoMatch(t *testing.T) {
	if got := Filter([]git.Branch{branch("main", 0)}, "zzz"); len(got) != 0 {
		t.Errorf("got %v", names(got))
	}
}
