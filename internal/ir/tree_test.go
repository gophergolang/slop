package ir

import "testing"

func TestEffectiveBasePathOverride(t *testing.T) {
	ent := &Entity{Name: "Task", API: API{BasePath: "/api/v1/tasks"}}
	if got := EffectiveBasePath(ent); got != "/api/v1/tasks" {
		t.Errorf("override path: got %q want /api/v1/tasks", got)
	}
}

func TestEffectiveBasePathDerivedNested(t *testing.T) {
	team := &Entity{Name: "Team"}
	task := &Entity{Name: "Task", ParentRefs: []*Entity{team}}
	got := EffectiveBasePath(task)
	want := "/api/v1/teams/:team_id/tasks"
	if got != want {
		t.Errorf("derived path: got %q want %q", got, want)
	}
}

func TestEffectiveBasePathRoot(t *testing.T) {
	team := &Entity{Name: "Team"}
	if got := EffectiveBasePath(team); got != "/api/v1/teams" {
		t.Errorf("root path: got %q want /api/v1/teams", got)
	}
}

func TestEntityRootPathChain(t *testing.T) {
	team := &Entity{Name: "Team"}
	task := &Entity{Name: "Task", ParentRefs: []*Entity{team}}
	comment := &Entity{Name: "Comment", ParentRefs: []*Entity{task}}
	chain := EntityRootPath(comment)
	if len(chain) != 3 || chain[0].Name != "Team" || chain[2].Name != "Comment" {
		t.Errorf("chain: got %v", names(chain))
	}
}

func names(es []*Entity) []string {
	out := make([]string, len(es))
	for i, e := range es {
		out[i] = e.Name
	}
	return out
}
