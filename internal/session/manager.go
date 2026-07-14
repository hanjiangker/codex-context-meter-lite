package session

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"codex-context-meter-lite/internal/meter"
)

type Manager struct {
	mu            sync.Mutex
	root          string
	indexPath     string
	pinnedID      string
	records       map[string]Summary
	titles        map[string]string
	indexMod      time.Time
	lastDiscovery time.Time
	knownPaths    []string
	selectedPath  string
	history       []meter.HistoryPoint
	tailer        Tailer
}

func NewManager(root, indexPath, pinnedID string) *Manager {
	return &Manager{root: root, indexPath: indexPath, pinnedID: normalizeThreadID(pinnedID), records: map[string]Summary{}, titles: map[string]string{}}
}

func (m *Manager) SetPinned(threadID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pinnedID = normalizeThreadID(threadID)
}

func (m *Manager) Refresh() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.reload(); err != nil {
		return State{Error: err.Error()}
	}
	return m.state()
}

func (m *Manager) reload() error {
	paths := m.knownPaths
	if len(paths) == 0 || time.Since(m.lastDiscovery) >= time.Second {
		var err error
		paths, err = ListSessionFiles(m.root, 2, 32)
		if err != nil {
			return err
		}
		m.knownPaths = paths
		m.lastDiscovery = time.Now()
	}
	m.reloadTitles()
	for _, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			continue
		}
		current, ok := m.records[path]
		if ok && current.FileSize == stat.Size() && current.FileModTime.Equal(stat.ModTime()) {
			continue
		}
		summary, err := ReadSummaryFast(path)
		if err != nil {
			continue
		}
		summary.Title = m.titleFor(summary)
		m.records[path] = summary
	}
	if m.pinnedID != "" && !m.hasThread(m.pinnedID) {
		if path, findErr := FindSessionFile(m.root, m.pinnedID); findErr == nil && path != "" {
			if summary, readErr := ReadSummaryFast(path); readErr == nil {
				summary.Title = m.titleFor(summary)
				m.records[path] = summary
			}
		}
	}

	selected := m.pickSelected()
	if selected == nil {
		m.selectedPath = ""
		m.history = nil
		return nil
	}
	if selected.Path != m.selectedPath {
		m.selectedPath = selected.Path
		events, _ := ReadHistory(selected.Path, time.Now().Add(-time.Hour))
		m.history = historyPoints(events)
		_ = m.tailer.Prime(selected.Path)
	} else {
		events, reset, err := m.tailer.ReadNew()
		if err == nil {
			if reset {
				events, _ = ReadHistory(selected.Path, time.Now().Add(-time.Hour))
				m.history = historyPoints(events)
			} else {
				m.appendHistory(events)
			}
		}
		if len(m.history) == 0 || m.history[len(m.history)-1].Time.Before(selected.UpdatedAt) {
			events, _ := ReadHistory(selected.Path, time.Now().Add(-time.Hour))
			m.history = historyPoints(events)
			_ = m.tailer.Prime(selected.Path)
		}
	}
	m.pruneHistory()
	return nil
}

func (m *Manager) hasThread(threadID string) bool {
	for _, item := range m.records {
		if normalizeThreadID(item.ThreadID) == normalizeThreadID(threadID) {
			return true
		}
	}
	return false
}

func (m *Manager) reloadTitles() {
	stat, err := os.Stat(m.indexPath)
	if err != nil || (!m.indexMod.IsZero() && stat.ModTime().Equal(m.indexMod)) {
		return
	}
	titles, err := LoadTitles(m.indexPath)
	if err == nil {
		m.titles = titles
		m.indexMod = stat.ModTime()
	}
}

func (m *Manager) titleFor(summary Summary) string {
	if title := m.titles[normalizeThreadID(summary.ThreadID)]; title != "" {
		return title
	}
	if summary.CWD != "" {
		return filepath.Base(summary.CWD)
	}
	id := normalizeThreadID(summary.ThreadID)
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func (m *Manager) pickSelected() *Summary {
	items := make([]Summary, 0, len(m.records))
	for _, item := range m.records {
		item.Title = m.titleFor(item)
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
	if m.pinnedID != "" {
		for i := range items {
			if normalizeThreadID(items[i].ThreadID) == m.pinnedID {
				copy := items[i]
				return &copy
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	copy := items[0]
	return &copy
}

func (m *Manager) state() State {
	selected := m.pickSelected()
	candidates := make([]Candidate, 0, len(m.records))
	seen := map[string]bool{}
	for _, item := range m.records {
		id := normalizeThreadID(item.ThreadID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		candidates = append(candidates, Candidate{ThreadID: id, Title: m.titleFor(item), UpdatedAt: item.UpdatedAt})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].UpdatedAt.After(candidates[j].UpdatedAt) })
	if len(candidates) > 12 {
		candidates = candidates[:12]
	}
	return State{Selected: selected, Candidates: candidates, History: append([]meter.HistoryPoint(nil), m.history...), Pinned: m.pinnedID != ""}
}

func historyPoints(events []TokenEvent) []meter.HistoryPoint {
	points := make([]meter.HistoryPoint, 0, len(events))
	for _, event := range events {
		var previous *meter.HistoryPoint
		if len(points) > 0 {
			previous = &points[len(points)-1]
		}
		points = append(points, meter.ToHistoryPoint(event.Time, event.Info, previous))
	}
	return points
}

func (m *Manager) appendHistory(events []TokenEvent) {
	for _, event := range events {
		var previous *meter.HistoryPoint
		if len(m.history) > 0 {
			previous = &m.history[len(m.history)-1]
		}
		m.history = append(m.history, meter.ToHistoryPoint(event.Time, event.Info, previous))
	}
}

func (m *Manager) pruneHistory() {
	cutoff := time.Now().Add(-time.Hour)
	index := sort.Search(len(m.history), func(i int) bool { return !m.history[i].Time.Before(cutoff) })
	if index > 0 {
		m.history = append([]meter.HistoryPoint(nil), m.history[index:]...)
	}
}

func DefaultPaths() (root, index string, err error) {
	codexHome := strings.TrimSpace(os.Getenv("CODEX_HOME"))
	if codexHome == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", "", homeErr
		}
		codexHome = filepath.Join(home, ".codex")
	}
	root, index = pathsFromCodexHome(codexHome)
	if _, statErr := os.Stat(root); statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return "", "", statErr
	}
	return root, index, nil
}

func ResolvePaths(sessionsOverride, indexOverride string) (root, index string, err error) {
	root, index, err = DefaultPaths()
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(sessionsOverride) != "" {
		root = filepath.Clean(sessionsOverride)
	}
	if strings.TrimSpace(indexOverride) != "" {
		index = filepath.Clean(indexOverride)
	}
	return root, index, nil
}

func pathsFromCodexHome(codexHome string) (root, index string) {
	base := filepath.Clean(codexHome)
	return filepath.Join(base, "sessions"), filepath.Join(base, "session_index.jsonl")
}
