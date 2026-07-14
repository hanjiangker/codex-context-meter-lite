package session

import (
	"bytes"
	"io"
	"os"
)

type Tailer struct {
	Path    string
	offset  int64
	partial []byte
}

func (t *Tailer) Prime(path string) error {
	t.Path = path
	t.offset = 0
	t.partial = nil
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	size := min(stat.Size(), int64(64*1024))
	start := stat.Size() - size
	data := make([]byte, size)
	if size > 0 {
		_, _ = file.ReadAt(data, start)
	}
	if index := bytes.LastIndexByte(data, '\n'); index >= 0 {
		t.offset = start + int64(index) + 1
		t.partial = append([]byte(nil), data[index+1:]...)
	} else {
		t.offset = stat.Size()
		t.partial = append([]byte(nil), data...)
	}
	return nil
}

func (t *Tailer) ReadNew() ([]TokenEvent, bool, error) {
	file, err := os.Open(t.Path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, false, err
	}
	reset := stat.Size() < t.offset
	if reset {
		if err := t.Prime(t.Path); err != nil {
			return nil, true, err
		}
		return nil, true, nil
	}
	if stat.Size() == t.offset {
		return nil, false, nil
	}
	if _, err := file.Seek(t.offset, io.SeekStart); err != nil {
		return nil, false, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, false, err
	}
	t.offset = stat.Size()
	data = append(t.partial, data...)
	lines := bytes.Split(data, []byte{'\n'})
	t.partial = append(t.partial[:0], lines[len(lines)-1]...)
	result := make([]TokenEvent, 0, len(lines)-1)
	for _, line := range lines[:len(lines)-1] {
		if event, ok := ParseTokenLine(line); ok {
			result = append(result, event)
		}
	}
	return result, false, nil
}
