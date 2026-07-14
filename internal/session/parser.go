package session

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"codex-context-meter-lite/internal/meter"
)

type envelope struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type sessionMeta struct {
	ID            string `json:"id"`
	CWD           string `json:"cwd"`
	ModelProvider string `json:"model_provider"`
	Timestamp     string `json:"timestamp"`
}

type eventPayload struct {
	Type string          `json:"type"`
	Info meter.TokenInfo `json:"info"`
}

type indexRow struct {
	ID         string `json:"id"`
	ThreadName string `json:"thread_name"`
}

func ParseTokenLine(line []byte) (TokenEvent, bool) {
	var row envelope
	if json.Unmarshal(bytes.TrimSpace(line), &row) != nil || row.Type != "event_msg" {
		return TokenEvent{}, false
	}
	var payload eventPayload
	if json.Unmarshal(row.Payload, &payload) != nil || payload.Type != "token_count" {
		return TokenEvent{}, false
	}
	at, err := time.Parse(time.RFC3339Nano, row.Timestamp)
	if err != nil || payload.Info.ModelContextWindow <= 0 {
		return TokenEvent{}, false
	}
	return TokenEvent{Time: at, Info: payload.Info}, true
}

func ReadSummaryFast(path string) (Summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return Summary{}, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return Summary{}, err
	}
	meta, err := readMeta(file)
	if err != nil {
		return Summary{}, err
	}
	event, err := readLatestToken(file)
	if err != nil {
		return Summary{}, err
	}
	created, _ := time.Parse(time.RFC3339Nano, meta.Timestamp)
	return Summary{
		Path: path, ThreadID: meta.ID, CWD: meta.CWD, Provider: meta.ModelProvider,
		CreatedAt: created, UpdatedAt: event.Time, Info: event.Info,
		FileSize: stat.Size(), FileModTime: stat.ModTime(),
	}, nil
}

func readMeta(file *os.File) (sessionMeta, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return sessionMeta{}, err
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for lines := 0; scanner.Scan() && lines < 256; lines++ {
		var row envelope
		if json.Unmarshal(scanner.Bytes(), &row) != nil || row.Type != "session_meta" {
			continue
		}
		var meta sessionMeta
		if json.Unmarshal(row.Payload, &meta) == nil && meta.ID != "" {
			return meta, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return sessionMeta{}, err
	}
	return sessionMeta{}, errors.New("session_meta not found")
}

func readLatestToken(file *os.File) (TokenEvent, error) {
	var found TokenEvent
	err := forEachLineReverse(file, func(line []byte) bool {
		if event, ok := ParseTokenLine(line); ok {
			found = event
			return false
		}
		return true
	})
	if err != nil {
		return TokenEvent{}, err
	}
	if found.Time.IsZero() {
		return TokenEvent{}, errors.New("token_count not found")
	}
	return found, nil
}

func forEachLineReverse(file *os.File, visit func([]byte) bool) error {
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	const chunkSize int64 = 256 * 1024
	position := stat.Size()
	buffer := []byte{}
	for position > 0 {
		size := min(chunkSize, position)
		position -= size
		chunk := make([]byte, size)
		if _, err := file.ReadAt(chunk, position); err != nil && err != io.EOF {
			return err
		}
		buffer = append(chunk, buffer...)
		lines := bytes.Split(buffer, []byte{'\n'})
		buffer = append([]byte(nil), lines[0]...)
		for i := len(lines) - 1; i >= 1; i-- {
			if len(bytes.TrimSpace(lines[i])) > 0 && !visit(lines[i]) {
				return nil
			}
		}
	}
	if len(bytes.TrimSpace(buffer)) > 0 {
		visit(buffer)
	}
	return nil
}

func ReadHistory(path string, cutoff time.Time) ([]TokenEvent, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	result := make([]TokenEvent, 0, 128)
	for scanner.Scan() {
		if event, ok := ParseTokenLine(scanner.Bytes()); ok && !event.Time.Before(cutoff) {
			result = append(result, event)
		}
	}
	return result, scanner.Err()
}

func LoadTitles(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	defer file.Close()
	titles := map[string]string{}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		var row indexRow
		if json.Unmarshal(scanner.Bytes(), &row) == nil && row.ID != "" && row.ThreadName != "" {
			titles[normalizeThreadID(row.ID)] = strings.TrimSpace(row.ThreadName)
		}
	}
	return titles, scanner.Err()
}

func ListSessionFiles(root string, recentDays, limit int) ([]string, error) {
	cutoff := time.Now().AddDate(0, 0, -recentDays)
	type item struct {
		path string
		mod  time.Time
	}
	items := make([]item, 0, limit)
	collect := func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".jsonl") {
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().After(cutoff) {
			items = append(items, item{path: path, mod: info.ModTime()})
		}
		return nil
	}
	foundDatedRoot := false
	for offset := 0; offset <= recentDays; offset++ {
		day := time.Now().AddDate(0, 0, -offset)
		dayRoot := filepath.Join(root, day.Format("2006"), day.Format("01"), day.Format("02"))
		if stat, err := os.Stat(dayRoot); err == nil && stat.IsDir() {
			foundDatedRoot = true
			if err := filepath.WalkDir(dayRoot, collect); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	if !foundDatedRoot {
		if err := filepath.WalkDir(root, collect); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].mod.After(items[j].mod) })
	if len(items) > limit {
		items = items[:limit]
	}
	paths := make([]string, len(items))
	for i := range items {
		paths[i] = items[i].path
	}
	return paths, nil
}

func FindSessionFile(root, threadID string) (string, error) {
	threadID = normalizeThreadID(threadID)
	if threadID == "" {
		return "", nil
	}
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".jsonl") && strings.Contains(entry.Name(), threadID) {
			found = path
			return io.EOF
		}
		return nil
	})
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return found, nil
}

func normalizeThreadID(value string) string {
	return strings.TrimPrefix(value, "local:")
}
