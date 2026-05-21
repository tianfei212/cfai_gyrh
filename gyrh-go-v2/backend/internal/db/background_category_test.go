package db

import "testing"

func newCategoryTestDB(t *testing.T) *DB {
	t.Helper()
	testDB, err := NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})
	return testDB
}

func createBackgroundPromptForCategoryTest(t *testing.T, repo *BackgroundPromptRepo, name string) int64 {
	t.Helper()
	id, err := repo.Create(name, "", "", "", "", "", "", "", "", "", "", "", "", "", "", 0, 0)
	if err != nil {
		t.Fatalf("create background prompt %q: %v", name, err)
	}
	return id
}

func TestBackgroundCategoryRepoEnsuresDefaultCategory(t *testing.T) {
	testDB := newCategoryTestDB(t)
	repo := NewBackgroundCategoryRepo(testDB)

	category, err := repo.EnsureDefault()
	if err != nil {
		t.Fatalf("ensure default: %v", err)
	}
	if category.ParentName != DefaultCategoryParent || category.ChildName != DefaultCategoryChild {
		t.Fatalf("default category = %q/%q", category.ParentName, category.ChildName)
	}

	again, err := repo.EnsureDefault()
	if err != nil {
		t.Fatalf("ensure default again: %v", err)
	}
	if again.ID != category.ID {
		t.Fatalf("default category duplicated: first=%d second=%d", category.ID, again.ID)
	}
}

func TestBackgroundCategoryRepoBackfillsUnboundBackgrounds(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "unbound")

	if err := categoryRepo.EnsureDefaultBindings(); err != nil {
		t.Fatalf("ensure default bindings: %v", err)
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].ParentName != DefaultCategoryParent || categories[0].ChildName != DefaultCategoryChild {
		t.Fatalf("categories = %+v, want default/default", categories)
	}
}

func TestBackgroundCategoryRepoSupportsManyToManyBindings(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	firstBackgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg-1")
	secondBackgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg-2")

	firstCategoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create first category: %v", err)
	}
	secondCategoryID, err := categoryRepo.Create("科幻电影", "机甲战斗")
	if err != nil {
		t.Fatalf("create second category: %v", err)
	}

	if err := categoryRepo.ReplaceBackgroundBindings(firstBackgroundID, []int64{firstCategoryID, secondCategoryID}); err != nil {
		t.Fatalf("bind first background: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(secondBackgroundID, []int64{firstCategoryID}); err != nil {
		t.Fatalf("bind second background: %v", err)
	}

	firstBackgroundCategories, err := categoryRepo.ListByBackgroundID(firstBackgroundID)
	if err != nil {
		t.Fatalf("list first background categories: %v", err)
	}
	if len(firstBackgroundCategories) != 2 {
		t.Fatalf("first background categories len = %d, want 2", len(firstBackgroundCategories))
	}

	firstCategoryBackgrounds, err := categoryRepo.ListBackgroundIDs(firstCategoryID)
	if err != nil {
		t.Fatalf("list first category backgrounds: %v", err)
	}
	if len(firstCategoryBackgrounds) != 2 {
		t.Fatalf("first category background len = %d, want 2", len(firstCategoryBackgrounds))
	}
}

func TestBackgroundCategoryRepoRejectsInvalidCategoryBindings(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg")

	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind background: %v", err)
	}

	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID + 999}); err == nil {
		t.Fatal("expected error binding nonexistent category")
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].ID != categoryID {
		t.Fatalf("categories after invalid replace = %+v, want original category", categories)
	}
}

func TestBackgroundCategoryRepoRejectsInvalidBackgroundBindings(t *testing.T) {
	testDB := newCategoryTestDB(t)
	categoryRepo := NewBackgroundCategoryRepo(testDB)

	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}

	if err := categoryRepo.ReplaceBackgroundBindings(999999, []int64{categoryID}); err == nil {
		t.Fatal("expected error binding nonexistent background")
	}

	backgroundIDs, err := categoryRepo.ListBackgroundIDs(categoryID)
	if err != nil {
		t.Fatalf("list category backgrounds: %v", err)
	}
	if len(backgroundIDs) != 0 {
		t.Fatalf("background IDs = %+v, want none", backgroundIDs)
	}
}

func TestBackgroundCategoryRepoProtectsDefaultCategory(t *testing.T) {
	testDB := newCategoryTestDB(t)
	repo := NewBackgroundCategoryRepo(testDB)
	defaultCategory, err := repo.EnsureDefault()
	if err != nil {
		t.Fatalf("ensure default: %v", err)
	}

	if err := repo.Update(defaultCategory.ID, "other", "other"); err == nil {
		t.Fatal("expected error updating default category")
	}
	if err := repo.Delete(defaultCategory.ID); err == nil {
		t.Fatal("expected error deleting default category")
	}
}

func TestBackgroundCategoryRepoDeleteKeepsExistingCategoryBinding(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg")

	firstCategoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create first category: %v", err)
	}
	secondCategoryID, err := categoryRepo.Create("科幻电影", "机甲战斗")
	if err != nil {
		t.Fatalf("create second category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{firstCategoryID, secondCategoryID}); err != nil {
		t.Fatalf("bind background: %v", err)
	}

	if err := categoryRepo.Delete(firstCategoryID); err != nil {
		t.Fatalf("delete category: %v", err)
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].ID != secondCategoryID {
		t.Fatalf("categories after delete = %+v, want remaining category", categories)
	}
}

func TestBackgroundCategoryRepoDeleteRebindsOrphansToDefault(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg")

	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind background: %v", err)
	}

	if err := categoryRepo.Delete(categoryID); err != nil {
		t.Fatalf("delete category: %v", err)
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 1 || categories[0].ParentName != DefaultCategoryParent || categories[0].ChildName != DefaultCategoryChild {
		t.Fatalf("categories after delete = %+v, want default/default", categories)
	}
}

func TestBackgroundCategoryRepoDeletesBackgroundBindings(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg")

	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind background: %v", err)
	}

	if err := categoryRepo.DeleteBackgroundBindings(backgroundID); err != nil {
		t.Fatalf("delete background bindings: %v", err)
	}

	categories, err := categoryRepo.ListByBackgroundID(backgroundID)
	if err != nil {
		t.Fatalf("list categories: %v", err)
	}
	if len(categories) != 0 {
		t.Fatalf("categories after cleanup = %+v, want none", categories)
	}

	backgroundIDs, err := categoryRepo.ListBackgroundIDs(categoryID)
	if err != nil {
		t.Fatalf("list category backgrounds: %v", err)
	}
	if len(backgroundIDs) != 0 {
		t.Fatalf("category background IDs after cleanup = %+v, want none", backgroundIDs)
	}
}

func TestBackgroundPromptRepoDeleteRemovesCategoryBindings(t *testing.T) {
	testDB := newCategoryTestDB(t)
	backgroundRepo := NewBackgroundPromptRepo(testDB)
	categoryRepo := NewBackgroundCategoryRepo(testDB)
	backgroundID := createBackgroundPromptForCategoryTest(t, backgroundRepo, "bg")

	categoryID, err := categoryRepo.Create("文旅片", "四川大佛")
	if err != nil {
		t.Fatalf("create category: %v", err)
	}
	if err := categoryRepo.ReplaceBackgroundBindings(backgroundID, []int64{categoryID}); err != nil {
		t.Fatalf("bind background: %v", err)
	}

	if err := backgroundRepo.Delete(backgroundID); err != nil {
		t.Fatalf("delete background prompt: %v", err)
	}

	backgroundIDs, err := categoryRepo.ListBackgroundIDs(categoryID)
	if err != nil {
		t.Fatalf("list category backgrounds: %v", err)
	}
	if len(backgroundIDs) != 0 {
		t.Fatalf("category background IDs after prompt delete = %+v, want none", backgroundIDs)
	}
}
