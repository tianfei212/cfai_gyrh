package db

import "testing"

func TestBackgroundPromptRepoListOrdersByNameDesc(t *testing.T) {
	database, err := NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create db: %v", err)
	}
	defer database.Close()

	repo := NewBackgroundPromptRepo(database)

	names := []string{"A 场景", "C 场景", "B 场景"}
	for _, name := range names {
		if _, err := repo.Create(name, "", "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0); err != nil {
			t.Fatalf("create background prompt %q: %v", name, err)
		}
	}

	items, err := repo.List(2, 0)
	if err != nil {
		t.Fatalf("list background prompts: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}

	got := []string{items[0].Name, items[1].Name}
	want := []string{"C 场景", "B 场景"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d name = %q, want %q; got all %v", i, got[i], want[i], got)
		}
	}
}
