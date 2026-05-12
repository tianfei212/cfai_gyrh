package db

import "testing"

func TestRewriteTaskRepoPersistsExternalTaskAndResult(t *testing.T) {
	testDB, err := NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	repo := NewRewriteTaskRepo(testDB)
	taskID := "rewrite_1"
	if err := repo.Create(RewriteTaskCreateInput{
		TaskID:             taskID,
		Provider:           "302-gpt-image",
		BackgroundPromptID: 7,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}
	if err := repo.SetExternalTaskID(taskID, "302-task-1"); err != nil {
		t.Fatalf("set external task id: %v", err)
	}
	if err := repo.Complete(taskID, RewriteTaskResult{
		ImageID:  27,
		AssetID:  "generated:rewrite_1.png",
		ImageURL: "https://example.com/rewrite_1.png",
	}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	task, err := repo.Get(taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if task.ExternalTaskID != "302-task-1" {
		t.Fatalf("external task id = %q", task.ExternalTaskID)
	}
	if task.Status != RewriteTaskStatusSucceeded {
		t.Fatalf("status = %q", task.Status)
	}
	if task.ImageID != 27 || task.AssetID == "" || task.ImageURL == "" {
		t.Fatalf("unexpected result fields: %+v", task)
	}
}

func TestRewriteTaskRepoListsRunningExternalTasks(t *testing.T) {
	testDB, err := NewDB(t.TempDir() + "/gyrh.db")
	if err != nil {
		t.Fatalf("create test db: %v", err)
	}
	t.Cleanup(func() {
		if err := testDB.Close(); err != nil {
			t.Fatalf("close test db: %v", err)
		}
	})

	repo := NewRewriteTaskRepo(testDB)
	if err := repo.Create(RewriteTaskCreateInput{TaskID: "running_with_external", Provider: "302-gpt-image"}); err != nil {
		t.Fatalf("create running task: %v", err)
	}
	if err := repo.SetExternalTaskID("running_with_external", "external-1"); err != nil {
		t.Fatalf("set external task id: %v", err)
	}
	if err := repo.Create(RewriteTaskCreateInput{TaskID: "running_without_external", Provider: "302-gpt-image"}); err != nil {
		t.Fatalf("create task without external id: %v", err)
	}
	if err := repo.Create(RewriteTaskCreateInput{TaskID: "completed", Provider: "302-gpt-image"}); err != nil {
		t.Fatalf("create completed task: %v", err)
	}
	if err := repo.SetExternalTaskID("completed", "external-2"); err != nil {
		t.Fatalf("set completed external task id: %v", err)
	}
	if err := repo.Complete("completed", RewriteTaskResult{ImageID: 1, AssetID: "asset", ImageURL: "url"}); err != nil {
		t.Fatalf("complete task: %v", err)
	}

	tasks, err := repo.ListRunningWithExternalTaskIDs()
	if err != nil {
		t.Fatalf("list running tasks: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("running tasks len = %d, want 1: %+v", len(tasks), tasks)
	}
	if tasks[0].TaskID != "running_with_external" || tasks[0].ExternalTaskID != "external-1" {
		t.Fatalf("unexpected task: %+v", tasks[0])
	}
}
