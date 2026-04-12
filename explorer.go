package main

import (
	"os"
	"path/filepath"
	"sort"
)

// ExplorerEntry represents a single file or directory in the explorer
type ExplorerEntry struct {
	Name  string
	Path  string
	IsDir bool
	Depth int
	Open  bool // if directory, whether it's expanded
}

// ExplorerModel holds the state for the file explorer panel
type ExplorerModel struct {
	Root     string
	Entries  []ExplorerEntry
	Selected int
	ScrollY  int
}

// NewExplorerModel creates a file explorer rooted at the given directory
func NewExplorerModel(root string) ExplorerModel {
	m := ExplorerModel{Root: root}
	m.Entries = scanDir(root, 0)
	return m
}

// scanDir reads directory entries, sorted: directories first, then alphabetical.
// Hidden files (starting with '.') are excluded.
func scanDir(dir string, depth int) []ExplorerEntry {
	dirEntries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	sort.Slice(dirEntries, func(i, j int) bool {
		di := dirEntries[i].IsDir()
		dj := dirEntries[j].IsDir()
		if di != dj {
			return di
		}
		return dirEntries[i].Name() < dirEntries[j].Name()
	})

	var result []ExplorerEntry
	for _, entry := range dirEntries {
		name := entry.Name()
		if len(name) > 0 && name[0] == '.' {
			continue
		}
		result = append(result, ExplorerEntry{
			Name:  name,
			Path:  filepath.Join(dir, name),
			IsDir: entry.IsDir(),
			Depth: depth,
		})
	}
	return result
}

// MoveUp moves the selection one entry up
func (m ExplorerModel) MoveUp() ExplorerModel {
	if m.Selected > 0 {
		m.Selected--
	}
	return m
}

// MoveDown moves the selection one entry down
func (m ExplorerModel) MoveDown() ExplorerModel {
	if m.Selected < len(m.Entries)-1 {
		m.Selected++
	}
	return m
}

// Toggle expands or collapses the selected directory
func (m ExplorerModel) Toggle() ExplorerModel {
	if m.Selected >= len(m.Entries) {
		return m
	}
	entry := m.Entries[m.Selected]
	if !entry.IsDir {
		return m
	}

	if entry.Open {
		// Collapse: remove all children (entries with greater depth)
		m.Entries[m.Selected].Open = false
		end := m.Selected + 1
		for end < len(m.Entries) && m.Entries[end].Depth > entry.Depth {
			end++
		}
		newEntries := make([]ExplorerEntry, 0, len(m.Entries)-(end-m.Selected-1))
		newEntries = append(newEntries, m.Entries[:m.Selected+1]...)
		newEntries = append(newEntries, m.Entries[end:]...)
		m.Entries = newEntries
	} else {
		// Expand: insert children after selected
		m.Entries[m.Selected].Open = true
		children := scanDir(entry.Path, entry.Depth+1)
		newEntries := make([]ExplorerEntry, 0, len(m.Entries)+len(children))
		newEntries = append(newEntries, m.Entries[:m.Selected+1]...)
		newEntries = append(newEntries, children...)
		newEntries = append(newEntries, m.Entries[m.Selected+1:]...)
		m.Entries = newEntries
	}
	return m
}

// SelectedEntry returns a pointer to the currently selected entry, or nil
func (m ExplorerModel) SelectedEntry() *ExplorerEntry {
	if m.Selected >= 0 && m.Selected < len(m.Entries) {
		return &m.Entries[m.Selected]
	}
	return nil
}

// ScrollToView adjusts scroll offset so the selected entry is visible
func (m ExplorerModel) ScrollToView(height int) ExplorerModel {
	if height <= 0 {
		return m
	}
	if m.Selected < m.ScrollY {
		m.ScrollY = m.Selected
	}
	if m.Selected >= m.ScrollY+height {
		m.ScrollY = m.Selected - height + 1
	}
	return m
}
