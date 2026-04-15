package gsm

import "testing"

func TestPushRecentProject(t *testing.T) {
	t.Run("adds to empty list", func(t *testing.T) {
		c := &gsmCache{}
		pushRecentProject(c, "proj-a")
		if len(c.RecentProjects) != 1 || c.RecentProjects[0] != "proj-a" {
			t.Errorf("got %v, want [proj-a]", c.RecentProjects)
		}
	})

	t.Run("moves existing to front", func(t *testing.T) {
		c := &gsmCache{RecentProjects: []string{"proj-a", "proj-b", "proj-c"}}
		pushRecentProject(c, "proj-c")
		want := []string{"proj-c", "proj-a", "proj-b"}
		if len(c.RecentProjects) != 3 {
			t.Fatalf("got len %d, want 3", len(c.RecentProjects))
		}
		for i, w := range want {
			if c.RecentProjects[i] != w {
				t.Errorf("index %d: got %q, want %q", i, c.RecentProjects[i], w)
			}
		}
	})

	t.Run("caps at maxRecentProj", func(t *testing.T) {
		c := &gsmCache{}
		for i := 0; i < maxRecentProj+5; i++ {
			pushRecentProject(c, "proj-"+string(rune('a'+i)))
		}
		if len(c.RecentProjects) != maxRecentProj {
			t.Errorf("got len %d, want %d", len(c.RecentProjects), maxRecentProj)
		}
	})

	t.Run("deduplicates", func(t *testing.T) {
		c := &gsmCache{RecentProjects: []string{"proj-a", "proj-b"}}
		pushRecentProject(c, "proj-a")
		if len(c.RecentProjects) != 2 {
			t.Errorf("got len %d, want 2", len(c.RecentProjects))
		}
		if c.RecentProjects[0] != "proj-a" {
			t.Errorf("got %q at 0, want proj-a", c.RecentProjects[0])
		}
	})
}
