package store

import (
	"errors"
	"sort"
	"sync"
)

type Flag struct {
	ID             int    `json:"id"`
	Key            string `json:"key"`
	Enabled        bool   `json:"enabled"`
	Description    string `json:"description"`
	RolloutPercent int    `json:"rollout_percent"`
}

var (
	ErrConflict = errors.New("flag already exists")
	ErrNotFound = errors.New("flag not found")
)

type Store struct {
	mu     sync.RWMutex
	flags  map[string]Flag
	nextID int
}

func New() *Store {
	return &Store{
		flags:  make(map[string]Flag),
		nextID: 1,
	}
}

func (s *Store) Create(key string, enabled bool, description string, rolloutPercent int) (Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.flags[key]; ok {
		return Flag{}, ErrConflict
	}

	f := Flag{
		ID:             s.nextID,
		Key:            key,
		Enabled:        enabled,
		Description:    description,
		RolloutPercent: rolloutPercent,
	}
	s.nextID++
	s.flags[key] = f
	return f, nil
}

func (s *Store) Get(key string) (Flag, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	f, ok := s.flags[key]
	if !ok {
		return Flag{}, ErrNotFound
	}
	return f, nil
}

func (s *Store) List() []Flag {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flags := make([]Flag, 0, len(s.flags))
	for _, f := range s.flags {
		flags = append(flags, f)
	}
	sort.Slice(flags, func(i, j int) bool {
		return flags[i].Key < flags[j].Key
	})
	return flags
}

func (s *Store) Update(key string, enabled *bool, description *string, rolloutPercent *int) (Flag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, ok := s.flags[key]
	if !ok {
		return Flag{}, ErrNotFound
	}

	if enabled != nil {
		f.Enabled = *enabled
	}
	if description != nil {
		f.Description = *description
	}
	if rolloutPercent != nil {
		f.RolloutPercent = *rolloutPercent
	}

	s.flags[key] = f
	return f, nil
}

func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.flags[key]; !ok {
		return ErrNotFound
	}
	delete(s.flags, key)
	return nil
}
