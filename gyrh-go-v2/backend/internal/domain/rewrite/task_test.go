package rewrite

import "testing"

func TestTaskCanCompleteFromRunning(t *testing.T) {
	task := Task{Status: StatusRunning}
	if err := task.MarkCompleted("asset-1"); err != nil {
		t.Fatalf("expected complete transition, got %v", err)
	}
	if task.Status != StatusCompleted || task.ResultAssetID != "asset-1" {
		t.Fatalf("unexpected task after completion: %+v", task)
	}
}

func TestTaskCannotCompleteFromFailed(t *testing.T) {
	task := Task{Status: StatusFailed}
	if err := task.MarkCompleted("asset-1"); err == nil {
		t.Fatal("expected invalid transition error")
	}
}
