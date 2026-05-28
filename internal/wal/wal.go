package wal

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

type Action string

const (
	ActionKeep    Action = "keep"
	ActionUnsub   Action = "unsubscribe"
	ActionDelete  Action = "delete"
	ActionHandled Action = "handled"
)

type walData struct {
	Keep        []string `toml:"keep"`
	Unsubscribe []string `toml:"unsubscribe"`
	Delete      []string `toml:"delete"`
	Handled     []string `toml:"handled"`
}

type WAL struct {
	mu   sync.RWMutex
	path string
	data walData
	idx  map[string]Action
}

// Load reads the WAL from path, creating an empty one if the file doesn't
// exist yet.
func Load(path string) (*WAL, error) {
	w := &WAL{path: path, idx: make(map[string]Action)}
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return w, nil
	}
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(raw, &w.data); err != nil {
		return nil, err
	}
	for _, addr := range w.data.Keep {
		w.idx[addr] = ActionKeep
	}
	for _, addr := range w.data.Unsubscribe {
		w.idx[addr] = ActionUnsub
	}
	for _, addr := range w.data.Delete {
		w.idx[addr] = ActionDelete
	}
	for _, addr := range w.data.Handled {
		w.idx[addr] = ActionHandled
	}
	return w, nil
}

// Mark records a decision for address and writes the file immediately.
// Calling Mark with an address already in the WAL moves it to the new action.
func (w *WAL) Mark(address string, action Action) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.data.Keep = removeStr(w.data.Keep, address)
	w.data.Unsubscribe = removeStr(w.data.Unsubscribe, address)
	w.data.Delete = removeStr(w.data.Delete, address)
	delete(w.idx, address)

	switch action {
	case ActionKeep:
		w.data.Keep = append(w.data.Keep, address)
		w.idx[address] = ActionKeep
	case ActionUnsub:
		w.data.Unsubscribe = append(w.data.Unsubscribe, address)
		w.idx[address] = ActionUnsub
	case ActionDelete:
		w.data.Delete = append(w.data.Delete, address)
		w.idx[address] = ActionDelete
	}
	return w.save()
}

// MarkExecuted moves addresses from the unsubscribe/delete sections into
// handled, persisting the change. Handled addresses remain in the index so
// the triage screen continues to filter them out.
func (w *WAL) MarkExecuted(addresses []string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, addr := range addresses {
		w.data.Unsubscribe = removeStr(w.data.Unsubscribe, addr)
		w.data.Delete = removeStr(w.data.Delete, addr)
		w.data.Handled = append(w.data.Handled, addr)
		w.idx[addr] = ActionHandled
	}
	return w.save()
}

func (w *WAL) IsDecided(address string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, ok := w.idx[address]
	return ok
}

func (w *WAL) ActionFor(address string) (Action, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	a, ok := w.idx[address]
	return a, ok
}

func (w *WAL) Counts() (keep, unsub, del int) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return len(w.data.Keep), len(w.data.Unsubscribe), len(w.data.Delete)
}

func (w *WAL) KeepAddresses() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return sliceCopy(w.data.Keep)
}

func (w *WAL) UnsubAddresses() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return sliceCopy(w.data.Unsubscribe)
}

func (w *WAL) DeleteAddresses() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return sliceCopy(w.data.Delete)
}

func (w *WAL) save() error {
	if err := os.MkdirAll(filepath.Dir(w.path), 0755); err != nil {
		return err
	}
	raw, err := toml.Marshal(w.data)
	if err != nil {
		return err
	}
	return os.WriteFile(w.path, raw, 0644)
}

func removeStr(s []string, v string) []string {
	out := s[:0]
	for _, e := range s {
		if e != v {
			out = append(out, e)
		}
	}
	return out
}

func sliceCopy(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	cp := make([]string, len(s))
	copy(cp, s)
	return cp
}
