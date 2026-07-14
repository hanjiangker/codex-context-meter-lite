package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadSummaryFastSkipsMalformedLines(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "session-basic.jsonl")
	summary, err := ReadSummaryFast(path)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ThreadID != "019f-demo-basic" || summary.Info.LastUsage.TotalTokens != 600 || summary.Info.ModelContextWindow != 1000 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestHistoryContainsCompactionEvents(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "session-compaction.jsonl")
	events, err := ReadHistory(path, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	points := historyPoints(events)
	if len(points) != 2 || !points[1].Reset {
		t.Fatalf("unexpected points: %+v", points)
	}
}

func TestTailerHoldsPartialLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rollout.jsonl")
	meta := `{"timestamp":"2026-07-13T09:00:00Z","type":"session_meta","payload":{"id":"tail"}}` + "\n"
	if err := os.WriteFile(path, []byte(meta), 0o600); err != nil {
		t.Fatal(err)
	}
	var tailer Tailer
	if err := tailer.Prime(path); err != nil {
		t.Fatal(err)
	}
	line := `{"timestamp":"2026-07-13T09:02:00Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"total_tokens":600},"last_token_usage":{"input_tokens":500,"output_tokens":100,"total_tokens":600},"model_context_window":1000}}}`
	half := len(line) / 2
	appendFile(t, path, line[:half])
	if events, _, err := tailer.ReadNew(); err != nil || len(events) != 0 {
		t.Fatalf("partial line should not emit: events=%v err=%v", events, err)
	}
	appendFile(t, path, line[half:]+"\n")
	events, reset, err := tailer.ReadNew()
	if err != nil || reset || len(events) != 1 || events[0].Info.LastUsage.TotalTokens != 600 {
		t.Fatalf("unexpected tail result: events=%v reset=%v err=%v", events, reset, err)
	}
}

func TestParseTokenLineRejectsMentionInMessage(t *testing.T) {
	line := `{"timestamp":"2026-07-13T09:02:00Z","type":"response_item","payload":{"type":"message","text":"token_count"}}`
	if _, ok := ParseTokenLine([]byte(line)); ok {
		t.Fatal("message text must not be treated as a token event")
	}
}

func TestReverseReaderIgnoresIncompleteLastLine(t *testing.T) {
	source := filepath.Join("..", "..", "testdata", "session-basic.jsonl")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	data = append(data, []byte(`{"timestamp":"2026-07-13T09:03:00Z","type":"event_msg"`)...)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	summary, err := ReadSummaryFast(path)
	if err != nil || summary.Info.LastUsage.TotalTokens != 600 {
		t.Fatalf("unexpected summary=%+v err=%v", summary, err)
	}
}

func TestLoadTitlesUsesLastValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session_index.jsonl")
	data := strings.Join([]string{
		`{"id":"abc","thread_name":"Old"}`,
		`{"id":"abc","thread_name":"New"}`,
	}, "\n")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	titles, err := LoadTitles(path)
	if err != nil || titles["abc"] != "New" {
		t.Fatalf("unexpected titles=%v err=%v", titles, err)
	}
}

func appendFile(t *testing.T, path, value string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(value); err != nil {
		t.Fatal(err)
	}
}
