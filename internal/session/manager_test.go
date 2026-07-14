package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerSelectsLatestTokenEventAndSupportsPin(t *testing.T) {
	root := t.TempDir()
	now := time.Now().UTC()
	oldPath := filepath.Join(root, "rollout-old.jsonl")
	newPath := filepath.Join(root, "rollout-new.jsonl")
	writeSession(t, oldPath, "old", now.Add(-2*time.Minute), 400)
	writeSession(t, newPath, "new", now.Add(-time.Minute), 600)
	if err := os.Chtimes(oldPath, now, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newPath, now.Add(-10*time.Minute), now.Add(-10*time.Minute)); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(root, "session_index.jsonl")
	if err := os.WriteFile(index, []byte("{\"id\":\"old\",\"thread_name\":\"Old task\"}\n{\"id\":\"new\",\"thread_name\":\"New task\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	manager := NewManager(root, index, "")
	state := manager.Refresh()
	if state.Error != "" || state.Selected == nil || state.Selected.ThreadID != "new" || state.Selected.Title != "New task" {
		t.Fatalf("unexpected auto state: %+v", state)
	}
	manager.SetPinned("old")
	state = manager.Refresh()
	if state.Selected == nil || state.Selected.ThreadID != "old" || !state.Pinned {
		t.Fatalf("unexpected pinned state: %+v", state)
	}
}

func TestManagerRebuildsHistoryAfterTruncateAndRewrite(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "rollout-rotated.jsonl")
	now := time.Now().UTC()
	writeSession(t, path, "rotated", now.Add(-time.Minute), 700)
	manager := NewManager(root, filepath.Join(root, "missing-index.jsonl"), "")
	state := manager.Refresh()
	if len(state.History) != 1 || state.History[0].UsedTokens != 700 {
		t.Fatalf("unexpected initial history: %+v", state.History)
	}

	writeSession(t, path, "rotated", now, 250)
	state = manager.Refresh()
	if state.Selected == nil || state.Selected.Info.LastUsage.TotalTokens != 250 {
		t.Fatalf("summary did not follow rewritten file: %+v", state.Selected)
	}
	if len(state.History) != 1 || state.History[0].UsedTokens != 250 {
		t.Fatalf("history was not rebuilt: %+v", state.History)
	}
}

func TestDefaultPathsPrefersCodexHome(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "custom-codex-home")
	t.Setenv("CODEX_HOME", codexHome)

	root, index, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}
	wantRoot, wantIndex := pathsFromCodexHome(codexHome)
	if root != wantRoot || index != wantIndex {
		t.Fatalf("DefaultPaths() = %q, %q; want %q, %q", root, index, wantRoot, wantIndex)
	}
}

func TestResolvePathsPrefersCLIOverrides(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "custom-codex-home")
	t.Setenv("CODEX_HOME", codexHome)
	sessionsOverride := filepath.Join(t.TempDir(), "sessions-override")
	indexOverride := filepath.Join(t.TempDir(), "index-override.jsonl")

	root, index, err := ResolvePaths(sessionsOverride, indexOverride)
	if err != nil {
		t.Fatal(err)
	}
	if root != filepath.Clean(sessionsOverride) || index != filepath.Clean(indexOverride) {
		t.Fatalf("ResolvePaths() = %q, %q", root, index)
	}
}

func TestResolvePathsAllowsIndependentOverrides(t *testing.T) {
	codexHome := filepath.Join(t.TempDir(), "custom-codex-home")
	t.Setenv("CODEX_HOME", codexHome)
	sessionsOverride := filepath.Join(t.TempDir(), "sessions-override")

	root, index, err := ResolvePaths(sessionsOverride, "")
	if err != nil {
		t.Fatal(err)
	}
	_, wantIndex := pathsFromCodexHome(codexHome)
	if root != filepath.Clean(sessionsOverride) || index != wantIndex {
		t.Fatalf("ResolvePaths() = %q, %q; want %q, %q", root, index, filepath.Clean(sessionsOverride), wantIndex)
	}
}

func writeSession(t *testing.T, path, id string, at time.Time, total int64) {
	t.Helper()
	rows := []map[string]any{
		{"timestamp": at.Add(-time.Second).Format(time.RFC3339Nano), "type": "session_meta", "payload": map[string]any{"id": id, "cwd": fmt.Sprintf(`D:\\%s`, id)}},
		{"timestamp": at.Format(time.RFC3339Nano), "type": "event_msg", "payload": map[string]any{"type": "token_count", "info": map[string]any{"total_token_usage": map[string]any{"total_tokens": total * 2}, "last_token_usage": map[string]any{"input_tokens": total - 10, "output_tokens": 10, "total_tokens": total}, "model_context_window": 1000}}},
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	for _, row := range rows {
		if err := encoder.Encode(row); err != nil {
			t.Fatal(err)
		}
	}
}
